//go:build OMIT

package main

import (
	"fmt"
	"strings"
)

func main() {
	// ટિક-ટૅક-ટોનું બોર્ડ બનાવો.
	board := [][]string{
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
	}

	// ખેલાડીઓ વારાફરતી ચાલ રમે છે.
	board[0][0] = "X"
	board[2][2] = "O"
	board[1][2] = "X"
	board[1][0] = "O"
	board[0][2] = "X"

	for i := 0; i < len(board); i++ {
		fmt.Printf("%s\n", strings.Join(board[i], " "))
	}
}
