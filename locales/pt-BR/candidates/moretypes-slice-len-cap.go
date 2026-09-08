//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Fatie o slice para que seu comprimento seja zero.
	s = s[:0]
	printSlice(s)

	// Aumente seu comprimento.
	s = s[:4]
	printSlice(s)

	// Descarte seus dois primeiros valores.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
