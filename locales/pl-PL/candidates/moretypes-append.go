//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append działa na wycinkach nil.
	s = append(s, 0)
	printSlice(s)

	// Wycinek rośnie w miarę potrzeb.
	s = append(s, 1)
	printSlice(s)

	// Można dodać więcej niż jeden element naraz.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
