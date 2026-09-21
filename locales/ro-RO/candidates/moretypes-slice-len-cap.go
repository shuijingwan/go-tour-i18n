//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Refă slice-ul astfel încât să aibă lungimea zero.
	s = s[:0]
	printSlice(s)

	// Extinde-i lungimea.
	s = s[:4]
	printSlice(s)

	// Elimină primele două valori.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
