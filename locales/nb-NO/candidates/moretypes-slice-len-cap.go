//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Lag en slice av slicen slik at lengden blir null.
	s = s[:0]
	printSlice(s)

	// Utvid lengden.
	s = s[:4]
	printSlice(s)

	// Fjern de to første verdiene.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
