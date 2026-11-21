package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// helpPanel is the key and mouse overview that is drawn on top of the board
type helpPanel struct {
	open   bool
	lines  []string
	scale  float32
	boardW int
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

// describe returns a short description of a keybindings.conf command, or "" for commands that are left out of the help
func describe(command string, args []string) string {
	arg := strings.Join(args, " ")
	switch command {
	case "left", "right", "up", "down":
		return "Move the cursor"
	case "playblack":
		return "Play black, GnuGo answers"
	case "playwhite":
		return "Play white, GnuGo answers"
	case "myplay":
		return "Play " + arg + " yourself"
	case "gnugowhite":
		return "GnuGo plays white"
	case "gnugoblack":
		return "GnuGo plays black"
	case "togglewhite":
		return "Put a white stone"
	case "toggleblack":
		return "Put a black stone"
	case "undo":
		return "Undo"
	case "passblack":
		return "Pass as black"
	case "passwhite":
		return "Pass as white"
	case "showlastmove":
		return "Show the last move"
	case "savesgf":
		return "Save the game"
	case "reloadsgf":
		return "Load the saved game"
	case "next":
		return "Next move"
	case "previous":
		return "Previous move"
	case "level":
		return "GnuGo strength"
	case "play100moves":
		return "GnuGo plays 100 moves"
	case "analyze":
		return "Analyze the position"
	case "gtp":
		switch arg {
		case "estimate_score":
			return "Estimate the score"
		case "final_score":
			return "Final score"
		}
	case "quit":
		return "Quit"
	case "clear":
		return "Clear the board"
	case "console":
		return "GTP console"
	case "save":
		return "Save a screenshot"
	case "status":
		return "Score and level"
	case "newasblack":
		return "New game as black"
	case "newaswhite":
		return "New game as white"
	}
	return ""
}

// keyLabel turns a keybindings.conf key name into the name shown in the help
func keyLabel(key string) string {
	switch strings.ToLower(key) {
	case "lmouse":
		return "Click"
	case "rmouse":
		return "Right click"
	case "mmouse":
		return "Middle click"
	}
	return key
}

// joinKeys lists the keys for one description, with runs of digits like 1, 2, ..., 9 written as 1-9
func joinKeys(keys []string) string {
	if slices.Equal(keys, []string{"Left", "Right", "Up", "Down"}) {
		return "Arrow keys"
	}
	var out []string
	for i := 0; i < len(keys); {
		j := i
		for j+1 < len(keys) && len(keys[j]) == 1 && len(keys[j+1]) == 1 && keys[j][0] >= '0' && keys[j][0] <= '9' && keys[j+1][0] == keys[j][0]+1 {
			j++
		}
		if j-i >= 2 {
			out = append(out, keys[i]+"-"+keys[j])
		} else {
			out = append(out, keys[i:j+1]...)
		}
		i = j + 1
	}
	return strings.Join(out, ", ")
}

// helpLines lists what the bindings in keybindings.conf do, one line per description, with the keys for it
func helpLines(confPath string) []string {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil
	}
	var order []string
	keys := map[string][]string{}
	for row := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(row)
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") || strings.HasPrefix(strings.ToLower(fields[0]), "kp") {
			continue
		}
		d := describe(fields[1], fields[2:])
		if d == "" {
			continue
		}
		if _, ok := keys[d]; !ok {
			order = append(order, d)
		}
		keys[d] = append(keys[d], keyLabel(fields[0]))
	}
	width := 0
	for _, d := range order {
		width = max(width, len(joinKeys(keys[d])))
	}
	width = min(width, 12)
	var out []string
	for _, d := range order {
		out = append(out, padRight(joinKeys(keys[d]), width)+"  "+d)
	}
	return append(out, "", padRight("h", width)+"  Show or hide this help")
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
	return 11 * h.scale
}

// box returns the area of the help text, centered on the board
func (h *helpPanel) box() (x, y, w, bh float32) {
	longest := 0
	for _, l := range h.lines {
		longest = max(longest, len(l))
	}
	w = float32(longest+4) * 8 * h.scale
	bh = float32(len(h.lines)+4) * h.lineH()
	return (float32(h.boardW) - w) / 2, (float32(h.boardW) - bh) / 2, w, bh
}

// button returns the area of the "?" button in the top right corner of the board
func (h *helpPanel) button() (x, y, w, bh float32) {
	w = 3 * 8 * h.scale
	bh = 2 * 8 * h.scale
	return float32(h.boardW) - w - 4*h.scale, 4 * h.scale, w, bh
}

func (h *helpPanel) hit(mx, my float32) bool {
	x, y, w, bh := h.button()
	return mx >= x && mx < x+w && my >= y && my < y+bh
}

func (h *helpPanel) draw(r *Renderer) {
	if h.open {
		x, y, w, bh := h.box()
		r.SetDrawColor(16, 18, 24, 230)
		r.FillRect(x, y, w, bh)
		r.SetDrawColor(0xff, 0xd8, 0x60, 255)
		r.DebugText(x+16*h.scale, y+h.lineH(), h.scale, "Monkeyjump Go")
		r.SetDrawColor(0xe0, 0xe0, 0xe0, 255)
		for i, l := range h.lines {
			r.DebugText(x+16*h.scale, y+float32(i+3)*h.lineH(), h.scale, l)
		}
	}
	x, y, w, bh := h.button()
	r.SetDrawColor(0, 0, 0, 160)
	r.FillRect(x, y, w, bh)
	r.SetDrawColor(0xe0, 0xe0, 0xe0, 255)
	label := " ?"
	if h.open {
		label = " x"
	}
	r.DebugText(x+4*h.scale, y+4*h.scale, h.scale, label)
}
