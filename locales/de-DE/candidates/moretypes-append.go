//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funktioniert mit nil-Slices.
	s = append(s, 0)
	printSlice(s)

	// Die Slice wächst nach Bedarf.
	s = append(s, 1)
	printSlice(s)

	// Wir können mehrere Elemente gleichzeitig hinzufügen.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
