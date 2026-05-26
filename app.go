package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// App ties everything together: window, renderer, board, GTP engine.
type App struct {
	window   *Window
	renderer *Renderer
	board    *Board
	gtp      *GTP
	br       *BoardRenderer
	filename string
}

func NewApp(boardSize int, filename string) (*App, error) {
	if err := sdlInit(); err != nil {
		return nil, err
	}

	title := "Monkeyjump Go"
	if filename != "" {
		title += " - " + filepath.Base(filename)
	}

	win, err := createWindow(title, windowWidth, windowHeight)
	if err != nil {
		sdlQuit()
		return nil, err
	}

	ren, err := createRenderer(win)
	if err != nil {
		win.Destroy()
		sdlQuit()
		return nil, err
	}

	// Read GnuGo command from config
	gtpCmd := "gnugo --mode gtp"
	confPath := filepath.Join("conf", "gnugocmd.conf")
	if data, err := os.ReadFile(confPath); err == nil {
		cmd := strings.TrimSpace(string(data))
		if cmd != "" {
			gtpCmd = cmd
		}
	}

	gtp, err := NewGTP(gtpCmd)
	if err != nil {
		ren.Destroy()
		win.Destroy()
		sdlQuit()
		return nil, fmt.Errorf("starting GTP engine: %w", err)
	}

	board := NewBoard(boardSize, gtp)

	// Read theme from config
	themeDir := filepath.Join("themes", "uligo")
	themeConfPath := filepath.Join("conf", "theme.conf")
	if data, err := os.ReadFile(themeConfPath); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				themeDir = filepath.Join("themes", line)
				break
			}
		}
	}

	br, err := NewBoardRenderer(ren, boardSize, windowWidth, windowHeight, themeDir)
	if err != nil {
		gtp.Close()
		ren.Destroy()
		win.Destroy()
		sdlQuit()
		return nil, err
	}

	app := &App{
		window:   win,
		renderer: ren,
		board:    board,
		gtp:      gtp,
		br:       br,
		filename: filename,
	}

	// Load SGF if provided
	if filename != "" {
		if err := board.LoadSGF(filename); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load SGF: %v\n", err)
		}
	}

	return app, nil
}

func (a *App) Destroy() {
	a.br.Destroy()
	a.gtp.Close()
	a.renderer.Destroy()
	a.window.Destroy()
	sdlQuit()
}

func (a *App) Run() {
	a.br.Draw(a.board)

	needsRedraw := false
	running := true
	for running {
		// Process all pending events before redrawing
		hadEvent := false
		for {
			ev, ok := pollEvent()
			if !ok {
				break
			}
			hadEvent = true

			switch ev.Type {
			case EventQuit:
				running = false

			case EventKeyDown:
				if a.handleKey(ev.Key) {
					running = false
				}
				needsRedraw = true

			case EventMouseButtonDown:
				x, y := a.br.ScreenToBoard(ev.MouseX, ev.MouseY)
				a.board.CursorX = x
				a.board.CursorY = y
				if ev.Button == 1 { // Left click: play black
					a.board.PlayBlackThenEngine(func() { a.br.Draw(a.board) })
				} else if ev.Button == 3 { // Right click: show status
					score := a.board.EstimateScore()
					fmt.Println("Score:", score)
				}
				needsRedraw = true

			case EventMouseMotion:
				x, y := a.br.ScreenToBoard(ev.MouseX, ev.MouseY)
				if x != a.board.CursorX || y != a.board.CursorY {
					a.board.CursorX = x
					a.board.CursorY = y
					needsRedraw = true
				}
			}
		}

		if needsRedraw {
			a.br.Draw(a.board)
			needsRedraw = false
		}

		if !hadEvent {
			sdlDelay(16) // ~60fps idle
		}
	}
}

// handleKey processes a key press. Returns true if the app should quit.
func (a *App) handleKey(scancode int) bool {
	draw := func() { a.br.Draw(a.board) }
	switch scancode {
	case ScancodeEscape, ScancodeQ:
		a.board.ShowBoard()
		return true
	case ScancodeLeft:
		a.board.MoveCursor(-1, 0)
	case ScancodeRight:
		a.board.MoveCursor(1, 0)
	case ScancodeUp:
		a.board.MoveCursor(0, -1)
	case ScancodeDown:
		a.board.MoveCursor(0, 1)
	case ScancodeSpace:
		a.board.PlayBlackThenEngine(draw)
	case ScancodeTab:
		a.board.PlayWhiteThenEngine(draw)
	case ScancodeReturn:
		a.board.GnuGoWhite()
	case ScancodeM:
		a.board.GnuGoBlack()
	case ScancodeW:
		a.board.ToggleWhite()
	case ScancodeB:
		a.board.ToggleBlack()
	case ScancodeU:
		a.board.Undo()
	case ScancodeA:
		a.board.PassBlack()
	case ScancodeZ:
		a.board.PassWhite()
	case ScancodeG:
		a.board.ShowLastMove()
	case ScancodeV:
		a.board.ShowBoard()
	case ScancodeE:
		score := a.board.EstimateScore()
		fmt.Println("Estimated score:", score)
	case ScancodeF:
		score := a.board.FinalScore()
		fmt.Println("Final score:", score)
	case ScancodeN:
		a.board.NextMove()
	case ScancodeP:
		a.board.PreviousMove()
	case ScancodeS:
		a.board.SaveSGF("saved.sgf")
	case ScancodeL:
		if a.filename != "" {
			a.board.LoadSGF(a.filename)
		}
	case ScancodeI:
		a.board.ToggleIllegal()
	case ScancodeBackspace:
		a.board.Clear()
	case Scancode0:
		a.board.SetLevel(10)
	case Scancode1:
		a.board.SetLevel(1)
	case Scancode2:
		a.board.SetLevel(2)
	case Scancode3:
		a.board.SetLevel(3)
	case Scancode4:
		a.board.SetLevel(4)
	case Scancode5:
		a.board.SetLevel(5)
	case Scancode6:
		a.board.SetLevel(6)
	case Scancode7:
		a.board.SetLevel(7)
	case Scancode8:
		a.board.SetLevel(8)
	case Scancode9:
		a.board.SetLevel(9)
	case ScancodeF1:
		a.board.gtp.Send("name")
		a.board.gtp.Send("version")
	}
	return false
}
