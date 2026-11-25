package main

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const statusDuration = 6 * time.Second

var (
	statusMut  sync.Mutex
	statusText string
	statusTime time.Time
)

// status prints a message and shows it in the bottom left corner of the window for a while
func status(format string, args ...any) {
	msg := strings.TrimSpace(fmt.Sprintf(format, args...))
	fmt.Println(msg)
	statusMut.Lock()
	statusText = strings.ReplaceAll(msg, "\n", " ")
	statusTime = time.Now()
	statusMut.Unlock()
}

// currentStatus returns the status message, if it should still be shown
func currentStatus() string {
	statusMut.Lock()
	defer statusMut.Unlock()
	if statusText == "" || time.Since(statusTime) > statusDuration {
		return ""
	}
	return statusText
}

// statusExpired returns true once when the shown status message has timed out
func statusExpired() bool {
	statusMut.Lock()
	defer statusMut.Unlock()
	if statusText != "" && time.Since(statusTime) > statusDuration {
		statusText = ""
		return true
	}
	return false
}

func drawStatus(r *Renderer, boardW, windowH int, scale float32) {
	text := currentStatus()
	if text == "" {
		return
	}
	pad := 4 * scale
	h := 8*scale + 2*pad
	maxChars := int((float32(boardW) - 2*pad) / (8 * scale))
	if maxChars > 0 && len(text) > maxChars {
		text = text[:maxChars]
	}
	w := float32(len(text))*8*scale + 2*pad
	r.SetDrawColor(0, 0, 0, 204)
	r.FillRect(0, float32(windowH)-h, w, h)
	r.SetDrawColor(0xe0, 0xe0, 0xe0, 255)
	r.DebugText(pad, float32(windowH)-h+pad, scale, text)
}

// drawThinking draws three black dots in the lower right corner while GnuGo is thinking
func drawThinking(r *Renderer, boardW, windowH int, scale float32) {
	radius := 2 * scale
	r.SetDrawColor(0, 0, 0, 255)
	for i := range 3 {
		cx := float32(boardW) - 6*scale - float32(2-i)*3*radius
		cy := float32(windowH) - 6*scale
		for dy := -radius; dy <= radius; dy++ {
			dx := float32(math.Sqrt(float64(radius*radius - dy*dy)))
			r.DrawLine(cx-dx, cy+dy, cx+dx, cy+dy)
		}
	}
}

// engineMoveText describes a pass or a resignation by GnuGo
func engineMoveText(color StoneColor, move string) string {
	name := "white"
	if color == Black {
		name = "black"
	}
	if move == "resign" {
		return "GnuGo resigns as " + name
	}
	return "GnuGo passes as " + name
}

// describeScore turns a GTP score like "W+12.5" or "B+Resign" into a sentence
func describeScore(score string, final bool) string {
	fields := strings.Fields(score)
	if len(fields) == 0 {
		return score
	}
	s := fields[0]
	if s == "0" {
		if final {
			return "The game is a draw"
		}
		return "The game is even"
	}
	color, margin, ok := strings.Cut(s, "+")
	if !ok || (color != "W" && color != "B") {
		return score
	}
	name := "White"
	if color == "B" {
		name = "Black"
	}
	switch {
	case strings.EqualFold(margin, "resign") || strings.EqualFold(margin, "r"):
		return name + " wins by resignation"
	case final:
		return name + " wins by " + margin
	default:
		return name + " leads by " + margin
	}
}
