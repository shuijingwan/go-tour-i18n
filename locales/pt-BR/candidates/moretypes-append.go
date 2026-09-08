//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funciona com slices nil.
	s = append(s, 0)
	printSlice(s)

	// O slice cresce conforme necessário.
	s = append(s, 1)
	printSlice(s)

	// Podemos adicionar mais de um elemento por vez.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
