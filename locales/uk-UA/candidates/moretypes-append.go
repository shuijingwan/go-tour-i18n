//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append працює з nil зрізами.
	s = append(s, 0)
	printSlice(s)

	// За потреби зріз збільшується.
	s = append(s, 1)
	printSlice(s)

	// За раз можна додати кілька елементів.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
