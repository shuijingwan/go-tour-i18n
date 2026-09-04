//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funciona con slices nil.
	s = append(s, 0)
	printSlice(s)

	// El slice crece según sea necesario.
	s = append(s, 1)
	printSlice(s)

	// Podemos añadir más de un elemento a la vez.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
