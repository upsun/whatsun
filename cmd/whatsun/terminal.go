package main

import (
	"os"

	"golang.org/x/term"
)

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd())) //nolint:gosec // G115: Fd() fits in int on all supported platforms
	if err != nil {
		return 80 // fallback width
	}
	return width
}
