//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append deluje na nil rezinah.
	s = append(s, 0)
	printSlice(s)

	// Rezina se po potrebi poveča.
	s = append(s, 1)
	printSlice(s)

	// Hkrati lahko dodamo več kot en element.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
