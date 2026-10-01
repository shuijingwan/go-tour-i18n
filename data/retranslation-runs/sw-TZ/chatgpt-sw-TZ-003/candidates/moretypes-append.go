//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append hufanya kazi kwenye vipande vya nil.
	s = append(s, 0)
	printSlice(s)

	// Kipande hukua inapohitajika.
	s = append(s, 1)
	printSlice(s)

	// Tunaweza kuongeza zaidi ya elementi moja kwa wakati mmoja.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
