//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funguje aj s nil rezmi.
	s = append(s, 0)
	printSlice(s)

	// Rez sa podľa potreby zväčšuje.
	s = append(s, 1)
	printSlice(s)

	// Naraz môžeme pridať viac než jeden prvok.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
