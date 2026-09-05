//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funziona sulle slice nil.
	s = append(s, 0)
	printSlice(s)

	// La slice cresce secondo necessità.
	s = append(s, 1)
	printSlice(s)

	// Possiamo aggiungere più di un elemento alla volta.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
