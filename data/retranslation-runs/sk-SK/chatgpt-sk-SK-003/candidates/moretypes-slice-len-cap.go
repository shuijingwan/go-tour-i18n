//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Orež rez tak, aby mal nulovú dĺžku.
	s = s[:0]
	printSlice(s)

	// Predĺž jeho dĺžku.
	s = s[:4]
	printSlice(s)

	// Odstráň jeho prvé dve hodnoty.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
