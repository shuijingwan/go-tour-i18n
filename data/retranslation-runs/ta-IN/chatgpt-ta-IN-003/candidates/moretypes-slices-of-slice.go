//go:build OMIT

package main

import (
	"fmt"
	"strings"
)

func main() {
	// tic-tac-toe பலகையை உருவாக்குங்கள்.
	board := [][]string{
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
	}

	// வீரர்கள் மாறி மாறி விளையாடுகிறார்கள்.
	board[0][0] = "X"
	board[2][2] = "O"
	board[1][2] = "X"
	board[1][0] = "O"
	board[0][2] = "X"

	for i := 0; i < len(board); i++ {
		fmt.Printf("%s\n", strings.Join(board[i], " "))
	}
}
