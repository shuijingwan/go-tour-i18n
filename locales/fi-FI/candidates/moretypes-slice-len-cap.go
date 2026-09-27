//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Muodosta viipaleesta nollan pituinen viipale.
	s = s[:0]
	printSlice(s)

	// Kasvata sen pituutta.
	s = s[:4]
	printSlice(s)

	// Poista sen kaksi ensimmäistä arvoa.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
