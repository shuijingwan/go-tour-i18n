//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append fungerar med nil-slices.
	s = append(s, 0)
	printSlice(s)

	// Slicen växer efter behov.
	s = append(s, 1)
	printSlice(s)

	// Vi kan lägga till fler än ett element åt gången.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
