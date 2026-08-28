//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Die Slice auf die Länge null zuschneiden.
	s = s[:0]
	printSlice(s)

	// Ihre Länge erweitern.
	s = s[:4]
	printSlice(s)

	// Ihre ersten beiden Werte entfernen.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
