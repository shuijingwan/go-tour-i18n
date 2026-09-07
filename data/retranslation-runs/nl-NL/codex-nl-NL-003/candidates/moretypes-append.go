//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append werkt op nil-slices.
	s = append(s, 0)
	printSlice(s)

	// De slice groeit wanneer dat nodig is.
	s = append(s, 1)
	printSlice(s)

	// We kunnen meer dan één element tegelijk toevoegen.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
