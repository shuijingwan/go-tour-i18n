//go:build OMIT

package main

import (
	"fmt"
	"strings"
)

func main() {
	// ടിക്-ടാക്-ടോ ബോർഡ് സൃഷ്ടിക്കുക.
	board := [][]string{
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
	}

	// കളിക്കാർ മാറിമാറി കളിക്കുന്നു.
	board[0][0] = "X"
	board[2][2] = "O"
	board[1][2] = "X"
	board[1][0] = "O"
	board[0][2] = "X"

	for i := 0; i < len(board); i++ {
		fmt.Printf("%s\n", strings.Join(board[i], " "))
	}
}
