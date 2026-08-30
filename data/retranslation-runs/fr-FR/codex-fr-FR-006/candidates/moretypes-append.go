//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append fonctionne avec les slices nil.
	s = append(s, 0)
	printSlice(s)

	// La slice s’agrandit selon les besoins.
	s = append(s, 1)
	printSlice(s)

	// Nous pouvons ajouter plusieurs éléments à la fois.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
