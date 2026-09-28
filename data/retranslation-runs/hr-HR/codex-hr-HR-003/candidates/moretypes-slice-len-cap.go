//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Postavi granice isječka tako da mu duljina bude nula.
	s = s[:0]
	printSlice(s)

	// Povećaj mu duljinu.
	s = s[:4]
	printSlice(s)

	// Odbaci prve dvije vrijednosti.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
