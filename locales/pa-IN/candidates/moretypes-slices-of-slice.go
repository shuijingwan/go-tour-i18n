//go:build OMIT

package main

import (
	"fmt"
	"strings"
)

func main() {
	// ਟਿਕ-ਟੈਕ-ਟੋ ਖੇਡ ਦਾ ਬੋਰਡ ਬਣਾਓ।
	board := [][]string{
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
	}

	// ਖਿਡਾਰੀ ਵਾਰੀ-ਵਾਰੀ ਚਾਲ ਚਲਦੇ ਹਨ।
	board[0][0] = "X"
	board[2][2] = "O"
	board[1][2] = "X"
	board[1][0] = "O"
	board[0][2] = "X"

	for i := 0; i < len(board); i++ {
		fmt.Printf("%s\n", strings.Join(board[i], " "))
	}
}
