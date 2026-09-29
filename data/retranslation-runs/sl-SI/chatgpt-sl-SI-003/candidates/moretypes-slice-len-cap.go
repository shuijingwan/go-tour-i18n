//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Odrežite rezino tako, da bo njena dolžina nič.
	s = s[:0]
	printSlice(s)

	// Povečajte njeno dolžino.
	s = s[:4]
	printSlice(s)

	// Odstranite njeni prvi dve vrednosti.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
