//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funcționează pe slice-uri nil.
	s = append(s, 0)
	printSlice(s)

	// Slice-ul crește atunci când este necesar.
	s = append(s, 1)
	printSlice(s)

	// Putem adăuga mai mult de un element odată.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
