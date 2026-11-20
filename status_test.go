package main

import "testing"

func TestDescribeScore(t *testing.T) {
	for _, c := range []struct {
		in    string
		final bool
		want  string
	}{
		{"W+12.5", true, "White wins by 12.5"},
		{"B+3.5 (upper bound: 5.5, lower: 1.5)", false, "Black leads by 3.5"},
		{"B+Resign", true, "Black wins by resignation"},
		{"0", true, "The game is a draw"},
		{"?", false, "?"},
	} {
		if got := describeScore(c.in, c.final); got != c.want {
			t.Errorf("describeScore(%q, %v) = %q, want %q", c.in, c.final, got, c.want)
		}
	}
}

func TestHelpLines(t *testing.T) {
	lines := helpLines("conf/keybindings.conf")
	if len(lines) < 10 || lines[0] != "Cursor movement" {
		t.Fatalf("unexpected help lines: %q", lines[:min(3, len(lines))])
	}
	for _, l := range lines {
		if len(l) > 0 && l[0] == '#' {
			t.Errorf("comment leaked into the help: %q", l)
		}
	}
}
