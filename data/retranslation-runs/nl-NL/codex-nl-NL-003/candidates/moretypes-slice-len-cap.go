//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Snijd de slice bij tot lengte nul.
	s = s[:0]
	printSlice(s)

	// Vergroot de lengte.
	s = s[:4]
	printSlice(s)

	// Laat de eerste twee waarden weg.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
