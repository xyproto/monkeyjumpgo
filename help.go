package main

import (
	"os"
	"path/filepath"
	"strings"
)

// helpPanel is the key and mouse overview that opens to the right of the board
type helpPanel struct {
	open    bool
	lines   []string
	scale   float32
	boardW  int
	width   int
	rows    int
	columnW float32
}

func helpSeenPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "monkeyjump", "help-seen")
}

// helpLines lists the bindings in keybindings.conf, with the comment above each group as a heading
func helpLines(confPath string) []string {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil
	}
	var out []string
	rows := strings.Split(string(data), "\n")
	for i, row := range rows {
		row = strings.TrimSpace(row)
		next := ""
		if i+1 < len(rows) {
			next = strings.TrimSpace(rows[i+1])
		}
		switch {
		case row == "":
		case strings.HasPrefix(row, "#"):
			if next != "" && !strings.HasPrefix(next, "#") {
				if len(out) > 0 {
					out = append(out, "")
				}
				heading := strings.TrimSpace(strings.TrimLeft(row, "#"))
				if i := strings.IndexAny(heading, ",("); i > 0 {
					heading = strings.TrimSpace(heading[:i])
				}
				out = append(out, heading)
			}
		default:
			fields := strings.Fields(row)
			out = append(out, "  "+padRight(fields[0], 11)+" "+strings.Join(fields[1:], " "))
		}
	}
	return out
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func newHelpPanel(confPath string, boardW int) *helpPanel {
	h := &helpPanel{lines: helpLines(confPath), boardW: boardW}
	h.scale = float32(max(1, boardW/450))
	longest := 0
	for _, l := range h.lines {
		longest = max(longest, len(l))
	}
	h.columnW = float32(longest+2) * 8 * h.scale
	h.rows = max(1, int((float32(boardW)-h.top()-4*h.scale)/h.lineH()))
	columns := (len(h.lines) + h.rows - 1) / h.rows
	h.width = int(float32(max(1, columns))*h.columnW + 16*h.scale)
	if p := helpSeenPath(); p != "" {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			h.open = true
			if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
				os.WriteFile(p, nil, 0o644)
			}
		}
	}
	return h
}

func (h *helpPanel) lineH() float32 {
	return 10 * h.scale
}

func (h *helpPanel) top() float32 {
	return 4*h.scale + 3*8*h.scale
}

func (h *helpPanel) windowWidth() int {
	if h.open {
		return h.boardW + h.width
	}
	return h.boardW
}

// button returns the area of the ">>" or "<<" button
func (h *helpPanel) button() (x, y, w, bh float32) {
	w = 3 * 8 * h.scale
	bh = 2 * 8 * h.scale
	x = float32(h.boardW) - w - 4*h.scale
	if h.open {
		x = float32(h.boardW) + 4*h.scale
	}
	return x, 4 * h.scale, w, bh
}

func (h *helpPanel) hit(mx, my float32) bool {
	x, y, w, bh := h.button()
	return mx >= x && mx < x+w && my >= y && my < y+bh
}

func (h *helpPanel) draw(r *Renderer, windowH int) {
	if h.open {
		r.SetDrawColor(24, 24, 28, 255)
		r.FillRect(float32(h.boardW), 0, float32(h.width), float32(windowH))
		r.SetDrawColor(0xe0, 0xe0, 0xe0, 255)
		for i, l := range h.lines {
			x := float32(h.boardW) + 8*h.scale + float32(i/h.rows)*h.columnW
			y := h.top() + float32(i%h.rows)*h.lineH()
			r.DebugText(x, y, h.scale, l)
		}
	}
	x, y, w, bh := h.button()
	r.SetDrawColor(0, 0, 0, 160)
	r.FillRect(x, y, w, bh)
	r.SetDrawColor(0xe0, 0xe0, 0xe0, 255)
	label := ">>"
	if h.open {
		label = "<<"
	}
	r.DebugText(x+4*h.scale, y+4*h.scale, h.scale, label)
}
