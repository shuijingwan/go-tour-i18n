//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Réduire la slice à une longueur nulle.
	s = s[:0]
	printSlice(s)

	// Augmenter sa longueur.
	s = s[:4]
	printSlice(s)

	// Supprimer ses deux premières valeurs.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
