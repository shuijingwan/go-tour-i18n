//go:build OMIT

package main

import (
	"fmt"
	"strings"
)

func main() {
	// একটি টিক-ট্যাক-টো বোর্ড তৈরি করুন।
	board := [][]string{
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
		[]string{"_", "_", "_"},
	}

	// খেলোয়াড়েরা পালা করে চাল দেয়।
	board[0][0] = "X"
	board[2][2] = "O"
	board[1][2] = "X"
	board[1][0] = "O"
	board[0][2] = "X"

	for i := 0; i < len(board); i++ {
		fmt.Printf("%s\n", strings.Join(board[i], " "))
	}
}
