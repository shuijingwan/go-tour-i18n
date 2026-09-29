//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Torna a segmentar el slice perquè tingui longitud zero.
	s = s[:0]
	printSlice(s)

	// Amplia’n la longitud.
	s = s[:4]
	printSlice(s)

	// Descarta’n els dos primers valors.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
