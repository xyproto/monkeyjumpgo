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
				} else if ev.Button == 2 { // Middle click: new game as black
					a.board.NewAsBlack(func() { a.br.Draw(a.board) })
				} else if ev.Button == 3 { // Right click: show status
					a.board.Status()
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

	// Cursor movement
	case ScancodeLeft:
		a.board.MoveCursor(-1, 0)
	case ScancodeRight:
		a.board.MoveCursor(1, 0)
	case ScancodeUp:
		a.board.MoveCursor(0, -1)
	case ScancodeDown:
		a.board.MoveCursor(0, 1)

	// Numpad jump positions (for 19x19 board positions)
	case ScancodeKP1:
		a.board.JumpPos(4, 16)
	case ScancodeKP2:
		a.board.JumpPos(10, 16)
	case ScancodeKP3:
		a.board.JumpPos(16, 16)
	case ScancodeKP4:
		a.board.JumpPos(4, 10)
	case ScancodeKP5:
		a.board.JumpPos(10, 10)
	case ScancodeKP6:
		a.board.JumpPos(16, 10)
	case ScancodeKP7:
		a.board.JumpPos(4, 4)
	case ScancodeKP8:
		a.board.JumpPos(10, 4)
	case ScancodeKP9:
		a.board.JumpPos(16, 4)

	// Placing stones and playing
	case ScancodeSpace:
		a.board.PlayBlackThenEngine(draw)
	case ScancodeTab:
		a.board.MyPlay("white", draw)
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

	// SGF-related
	case ScancodeS:
		a.board.SaveSGF("saved.sgf")
	case ScancodeL:
		if a.board.filename != "" {
			a.board.LoadSGF(a.board.filename)
		}
	case ScancodeN:
		a.board.NextMove()
	case ScancodeP:
		a.board.PreviousMove()

	// GnuGo commands
	case ScancodeF1:
		a.board.GnuGoInfo()
	case ScancodeV:
		a.board.ShowBoard()
	case ScancodeKP0:
		a.board.SetLevel(0)
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
	case Scancode0:
		a.board.SetLevel(10)
	case ScancodeT:
		a.board.TimeSettings(5, 0, 0)
	case ScancodeY:
		a.board.Play100Moves(draw)
	case ScancodeO:
		a.board.FutureWhite()

	// GTP commands
	case ScancodeE:
		score := a.board.EstimateScore()
		fmt.Println("Estimated score:", score)
	case ScancodeF:
		score := a.board.FinalScore()
		fmt.Println("Final score:", score)
	case ScancodeX:
		a.board.MultiGTP("move_probabilities")
	case ScancodeF3:
		a.board.MultiGTP("dragon_status")
		a.board.MultiGTP("show_dragons")
	case ScancodeF2:
		// Analyze: use liberty-based heuristic to suggest and play a move
		a.board.Analyze(draw)

	// Special functions
	case ScancodeI:
		a.board.ToggleIllegal()
	case ScancodeBackspace:
		a.board.Clear()
	case ScancodeC:
		a.board.Console()
	case ScancodeF12:
		if err := a.renderer.SaveScreenshot("goboard.bmp"); err != nil {
			fmt.Fprintf(os.Stderr, "Screenshot failed: %v\n", err)
		} else {
			fmt.Println("Saved: goboard.bmp")
		}
	}
	return false
}
