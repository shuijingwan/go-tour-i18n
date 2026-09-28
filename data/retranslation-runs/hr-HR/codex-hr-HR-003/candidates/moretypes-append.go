//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append radi s isječcima nil.
	s = append(s, 0)
	printSlice(s)

	// Isječak raste prema potrebi.
	s = append(s, 1)
	printSlice(s)

	// Možemo dodati više elemenata odjednom.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
