//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append toimii myös nil-viipaleilla.
	s = append(s, 0)
	printSlice(s)

	// Viipale kasvaa tarpeen mukaan.
	s = append(s, 1)
	printSlice(s)

	// Voimme lisätä useita alkioita kerralla.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
