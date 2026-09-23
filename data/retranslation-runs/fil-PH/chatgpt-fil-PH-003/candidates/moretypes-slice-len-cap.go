//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// I-slice ang slice upang maging zero ang haba nito.
	s = s[:0]
	printSlice(s)

	// Palawakin ang haba nito.
	s = s[:4]
	printSlice(s)

	// Alisin ang unang dalawang value nito.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
