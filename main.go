package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultBoardSize = 9
	windowWidth      = 512
	windowHeight     = 512
)

func main() {
	boardSize := defaultBoardSize
	var filename string

	if len(os.Args) > 1 {
		arg := strings.Join(os.Args[1:], " ")
		if n, err := strconv.Atoi(arg); err == nil {
			boardSize = n
		} else {
			filename = arg
		}
	}

	if filename != "" {
		data, err := os.ReadFile(filename)
		if err == nil {
			s := string(data)
			switch {
			case strings.Contains(s, "SZ[9]"):
				boardSize = 9
			case strings.Contains(s, "SZ[13]"):
				boardSize = 13
			case strings.Contains(s, "SZ[19]"):
				boardSize = 19
			}
		}
	}

	app, err := NewApp(boardSize, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer app.Destroy()

	app.Run()
}
