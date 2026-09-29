//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funciona amb nil en el cas dels slices.
	s = append(s, 0)
	printSlice(s)

	// El slice creix segons calgui.
	s = append(s, 1)
	printSlice(s)

	// Podem afegir més d’un element alhora.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
