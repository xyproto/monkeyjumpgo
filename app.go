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
	kb       *Keybindings
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
	confPath := dataPath("conf", "gnugocmd.conf")
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
	themeDir := dataPath("themes", "uligo")
	themeConfPath := dataPath("conf", "theme.conf")
	if data, err := os.ReadFile(themeConfPath); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				themeDir = dataPath("themes", line)
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
		kb:       LoadKeybindings(dataPath("conf")),
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
				if binding, ok := a.kb.Mouse[ev.Button]; ok {
					if a.ExecuteBinding(binding, func() { a.br.Draw(a.board) }) {
						running = false
					}
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
	if binding, ok := a.kb.Keys[scancode]; ok {
		return a.ExecuteBinding(binding, func() { a.br.Draw(a.board) })
	}
	return false
}
