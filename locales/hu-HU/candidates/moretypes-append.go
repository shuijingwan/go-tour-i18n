//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// Az append nil szeleteken működik.
	s = append(s, 0)
	printSlice(s)

	// A szelet szükség szerint növekszik.
	s = append(s, 1)
	printSlice(s)

	// Egyszerre több elemet is hozzáadhatunk.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
