//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append darbojas ar nil šķēlumiem.
	s = append(s, 0)
	printSlice(s)

	// Šķēlums pēc vajadzības aug.
	s = append(s, 1)
	printSlice(s)

	// Vienā reizē varam pievienot vairāk nekā vienu elementu.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
