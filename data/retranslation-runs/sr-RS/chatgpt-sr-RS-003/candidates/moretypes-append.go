//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append ради над nil исечцима.
	s = append(s, 0)
	printSlice(s)

	// Исечак расте по потреби.
	s = append(s, 1)
	printSlice(s)

	// Можемо додати више од једног елемента одједном.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
