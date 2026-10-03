//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Verkürze den Slice auf die Länge null.
	s = s[:0]
	printSlice(s)

	// Vergrössere seine Länge.
	s = s[:4]
	printSlice(s)

	// Entferne seine ersten beiden Werte.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
