//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append veikia su nil pjūviais.
	s = append(s, 0)
	printSlice(s)

	// Prireikus pjūvis didėja.
	s = append(s, 1)
	printSlice(s)

	// Vienu metu galime pridėti daugiau nei vieną elementą.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
