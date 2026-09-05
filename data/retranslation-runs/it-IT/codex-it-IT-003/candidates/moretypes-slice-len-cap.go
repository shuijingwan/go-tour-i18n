//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Riduci la slice a una lunghezza pari a zero.
	s = s[:0]
	printSlice(s)

	// Estendi la sua lunghezza.
	s = s[:4]
	printSlice(s)

	// Elimina i primi due valori.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
