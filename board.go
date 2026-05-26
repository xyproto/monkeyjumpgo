package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Stone color
type StoneColor int

const (
	Empty StoneColor = iota
	Black
	White
)

// Board manages the Go board state and communicates with GnuGo.
type Board struct {
	Size       int
	Stones     map[[2]int]StoneColor
	CursorX    int
	CursorY    int
	oldCursorX int
	oldCursorY int
	gtp        *GTP
	bCaptures  int
	wCaptures  int
	lastMove   [2]int
	hasLast    bool
	level      int
	illegalOK  bool
	// SGF replay
	history      []string // e.g. "BD4", "WQ16"
	historyIndex int
}

// Letters used for Go coordinates (skipping 'I')
var letters = []byte("ABCDEFGHJKLMNOPQRST")

func NewBoard(size int, gtp *GTP) *Board {
	b := &Board{
		Size:         size,
		Stones:       make(map[[2]int]StoneColor),
		gtp:          gtp,
		level:        9,
		historyIndex: -1,
	}
	gtp.Send(fmt.Sprintf("boardsize %d", size))
	gtp.Send("clear_board")
	b.CursorX = size / 2
	b.CursorY = size / 2
	b.oldCursorX = b.CursorX
	b.oldCursorY = b.CursorY
	return b
}

// NumToGTP converts numeric board coords (0-based) to GTP format like "D4".
func (b *Board) NumToGTP(x, y int) string {
	if x < 0 || x >= b.Size || y < 0 || y >= b.Size {
		return ""
	}
	letter := letters[x]
	number := b.Size - y
	return fmt.Sprintf("%c%d", letter, number)
}

// GTPToNum converts GTP position like "D4" to numeric coords (0-based).
func (b *Board) GTPToNum(pos string) (int, int, error) {
	pos = strings.TrimSpace(strings.ToUpper(pos))
	if len(pos) < 2 {
		return 0, 0, fmt.Errorf("invalid position: %s", pos)
	}
	letter := pos[0]
	x := -1
	for i, l := range letters[:b.Size] {
		if l == letter {
			x = i
			break
		}
	}
	if x < 0 {
		return 0, 0, fmt.Errorf("invalid column: %c", letter)
	}
	num, err := strconv.Atoi(pos[1:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row: %s", pos[1:])
	}
	y := b.Size - num
	if y < 0 || y >= b.Size {
		return 0, 0, fmt.Errorf("row out of range: %d", num)
	}
	return x, y, nil
}

// SGFToNum converts SGF coordinates like "dp" to numeric coords.
// In standard SGF, 'a'=0, 'b'=1, etc. The y-axis goes top to bottom.
// When withI is true, the alphabet includes 'i'. When false, 'i' is skipped
// (matching GTP column convention), but for SGF this is non-standard.
func (b *Board) SGFToNum(sgf string, withI bool) (int, int, error) {
	if len(sgf) < 2 {
		return 0, 0, fmt.Errorf("invalid SGF coord: %s", sgf)
	}
	var alpha string
	if withI {
		alpha = "abcdefghijklmnopqrs"
	} else {
		alpha = "abcdefghjklmnopqrst"
	}
	x := strings.IndexByte(alpha, sgf[0])
	y := strings.IndexByte(alpha, sgf[1])
	if x < 0 || y < 0 {
		return 0, 0, fmt.Errorf("invalid SGF coord: %s", sgf)
	}
	if x >= b.Size || y >= b.Size {
		return 0, 0, fmt.Errorf("SGF coord out of range: %s", sgf)
	}
	return x, y, nil
}

func (b *Board) MoveCursor(dx, dy int) {
	b.oldCursorX = b.CursorX
	b.oldCursorY = b.CursorY
	b.CursorX += dx
	if b.CursorX < 0 {
		b.CursorX = b.Size - 1
	} else if b.CursorX >= b.Size {
		b.CursorX = 0
	}
	b.CursorY += dy
	if b.CursorY < 0 {
		b.CursorY = b.Size - 1
	} else if b.CursorY >= b.Size {
		b.CursorY = 0
	}
}

// PlayBlackThenEngine places a black stone and asks GnuGo to respond as white.
func (b *Board) PlayBlackThenEngine(drawFn func()) {
	pos := b.NumToGTP(b.CursorX, b.CursorY)
	if pos == "" {
		return
	}
	resp := b.gtp.Send(fmt.Sprintf("play black %s", pos))
	if strings.Contains(resp, "illegal") || strings.HasPrefix(resp, "?") {
		if !b.illegalOK {
			fmt.Println("Illegal move:", pos)
			return
		}
		fmt.Println("Illegal move (forced):", pos)
	}
	b.Stones[[2]int{b.CursorX, b.CursorY}] = Black
	b.lastMove = [2]int{b.CursorX, b.CursorY}
	b.hasLast = true
	b.checkCaptures()
	// Show the player's stone before engine thinks
	if drawFn != nil {
		drawFn()
	}
	b.GnuGoWhite()
}

// PlayWhiteThenEngine places a white stone and asks GnuGo to respond as black.
func (b *Board) PlayWhiteThenEngine(drawFn func()) {
	pos := b.NumToGTP(b.CursorX, b.CursorY)
	if pos == "" {
		return
	}
	resp := b.gtp.Send(fmt.Sprintf("play white %s", pos))
	if strings.Contains(resp, "illegal") || strings.HasPrefix(resp, "?") {
		if !b.illegalOK {
			fmt.Println("Illegal move:", pos)
			return
		}
	}
	b.Stones[[2]int{b.CursorX, b.CursorY}] = White
	b.lastMove = [2]int{b.CursorX, b.CursorY}
	b.hasLast = true
	b.checkCaptures()
	// Show the player's stone before engine thinks
	if drawFn != nil {
		drawFn()
	}
	b.GnuGoBlack()
}

// ToggleBlack places a black stone at cursor (no engine response).
func (b *Board) ToggleBlack() bool {
	pos := b.NumToGTP(b.CursorX, b.CursorY)
	resp := b.gtp.Send(fmt.Sprintf("play black %s", pos))
	if (strings.Contains(resp, "illegal") || strings.HasPrefix(resp, "?")) && !b.illegalOK {
		return false
	}
	b.Stones[[2]int{b.CursorX, b.CursorY}] = Black
	b.lastMove = [2]int{b.CursorX, b.CursorY}
	b.hasLast = true
	b.checkCaptures()
	return true
}

// ToggleWhite places a white stone at cursor (no engine response).
func (b *Board) ToggleWhite() bool {
	pos := b.NumToGTP(b.CursorX, b.CursorY)
	resp := b.gtp.Send(fmt.Sprintf("play white %s", pos))
	if (strings.Contains(resp, "illegal") || strings.HasPrefix(resp, "?")) && !b.illegalOK {
		return false
	}
	b.Stones[[2]int{b.CursorX, b.CursorY}] = White
	b.lastMove = [2]int{b.CursorX, b.CursorY}
	b.hasLast = true
	b.checkCaptures()
	return true
}

func (b *Board) GnuGoWhite() {
	move := b.gtp.SendExtract("genmove white")
	move = strings.TrimSpace(move)
	if move == "PASS" || move == "resign" {
		fmt.Println("GnuGo (white):", move)
		return
	}
	b.placeGTPMove(White, move)
}

func (b *Board) GnuGoBlack() {
	move := b.gtp.SendExtract("genmove black")
	move = strings.TrimSpace(move)
	if move == "PASS" || move == "resign" {
		fmt.Println("GnuGo (black):", move)
		return
	}
	b.placeGTPMove(Black, move)
}

func (b *Board) placeGTPMove(color StoneColor, pos string) {
	x, y, err := b.GTPToNum(pos)
	if err != nil {
		fmt.Printf("Error parsing GTP move %q: %v\n", pos, err)
		return
	}
	b.Stones[[2]int{x, y}] = color
	b.lastMove = [2]int{x, y}
	b.hasLast = true
	b.CursorX = x
	b.CursorY = y
	b.checkCaptures()
	fmt.Printf("GnuGo plays: %s (%d,%d)\n", pos, x, y)
}

// checkCaptures queries the engine for captures and resyncs the board if any occurred.
func (b *Board) checkCaptures() {
	bResp := b.gtp.SendExtract("captures black")
	wResp := b.gtp.SendExtract("captures white")
	bc, _ := strconv.Atoi(strings.TrimSpace(bResp))
	wc, _ := strconv.Atoi(strings.TrimSpace(wResp))
	if bc != b.bCaptures || wc != b.wCaptures {
		b.bCaptures = bc
		b.wCaptures = wc
		b.syncFromEngine()
	}
}

// syncFromEngine reloads the board state from GnuGo.
func (b *Board) syncFromEngine() {
	b.Stones = make(map[[2]int]StoneColor)
	blackStones := b.gtp.SendExtract("list_stones black")
	whiteStones := b.gtp.SendExtract("list_stones white")
	for pos := range strings.FieldsSeq(blackStones) {
		if x, y, err := b.GTPToNum(pos); err == nil {
			b.Stones[[2]int{x, y}] = Black
		}
	}
	for pos := range strings.FieldsSeq(whiteStones) {
		if x, y, err := b.GTPToNum(pos); err == nil {
			b.Stones[[2]int{x, y}] = White
		}
	}
}

func (b *Board) Undo() {
	// Undo the engine's last move and the player's last move
	resp := b.gtp.Send("undo")
	if strings.Contains(resp, "cannot") {
		return
	}
	// Try to undo the player's move too (undo a full turn)
	resp2 := b.gtp.Send("undo")
	if strings.Contains(resp2, "cannot") {
		// Only one move to undo, that's fine
	}
	b.syncFromEngine()
}

func (b *Board) PassBlack() {
	b.gtp.Send("play black PASS")
	b.GnuGoWhite()
}

func (b *Board) PassWhite() {
	b.gtp.Send("play white PASS")
	b.GnuGoBlack()
}

func (b *Board) Clear() {
	b.Stones = make(map[[2]int]StoneColor)
	b.gtp.Send("clear_board")
	b.bCaptures = 0
	b.wCaptures = 0
	b.history = nil
	b.historyIndex = -1
	b.CursorX = b.Size / 2
	b.CursorY = b.Size / 2
}

func (b *Board) ShowBoard() {
	b.gtp.Send("showboard")
}

func (b *Board) EstimateScore() string {
	return b.gtp.SendExtract("estimate_score")
}

func (b *Board) FinalScore() string {
	return b.gtp.SendExtract("final_score")
}

func (b *Board) SetLevel(level int) {
	b.level = level
	b.gtp.Send(fmt.Sprintf("level %d", level))
}

func (b *Board) ShowLastMove() {
	if b.hasLast {
		b.CursorX = b.lastMove[0]
		b.CursorY = b.lastMove[1]
	}
}

func (b *Board) ToggleIllegal() {
	b.illegalOK = !b.illegalOK
	if b.illegalOK {
		fmt.Println("Illegal moves: allowed")
	} else {
		fmt.Println("Illegal moves: blocked")
	}
}

func (b *Board) SaveSGF(filename string) {
	b.gtp.Send(fmt.Sprintf("printsgf %s", filename))
	fmt.Println("Saved:", filename)
}

// NextMove advances in the SGF history.
func (b *Board) NextMove() {
	if len(b.history) == 0 {
		return
	}
	b.historyIndex++
	if b.historyIndex >= len(b.history) {
		b.historyIndex = len(b.history) - 1
		return
	}
	colorPos := b.history[b.historyIndex]
	if len(colorPos) < 2 {
		return
	}
	color := string(colorPos[0])
	pos := colorPos[1:]
	if color == "B" {
		b.gtp.Send(fmt.Sprintf("play black %s", pos))
	} else {
		b.gtp.Send(fmt.Sprintf("play white %s", pos))
	}
	x, y, err := b.GTPToNum(pos)
	if err == nil {
		if color == "B" {
			b.Stones[[2]int{x, y}] = Black
		} else {
			b.Stones[[2]int{x, y}] = White
		}
		b.CursorX = x
		b.CursorY = y
		b.lastMove = [2]int{x, y}
		b.hasLast = true
	}
	b.checkCaptures()
}

// PreviousMove goes back in the SGF history.
func (b *Board) PreviousMove() {
	if b.historyIndex < 0 {
		return
	}
	b.historyIndex--
	b.gtp.Send("undo")
	b.syncFromEngine()
}
