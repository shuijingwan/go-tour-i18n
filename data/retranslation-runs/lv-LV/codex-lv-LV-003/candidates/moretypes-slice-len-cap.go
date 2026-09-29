//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Izveido no šķēluma šķēlumu ar nulles garumu.
	s = s[:0]
	printSlice(s)

	// Palielina tā garumu.
	s = s[:4]
	printSlice(s)

	// Atmet tā pirmās divas vērtības.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
