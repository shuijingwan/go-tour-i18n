//go:build OMIT

package main

import (
	"fmt"
	"strings"
)

func main() {
	// Sukurkite kryžiukų ir nuliukų žaidimo lentą.
	board := [][]string{
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
	}

	// Žaidėjai atlieka ėjimus paeiliui.
	board[0][0] = "X"
	board[2][2] = "O"
	board[1][2] = "X"
	board[1][0] = "O"
	board[0][2] = "X"

	for i := 0; i < len(board); i++ {
		fmt.Printf("%s\n", strings.Join(board[i], " "))
	}
}
