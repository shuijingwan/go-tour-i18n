//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Skapa en slice av slicen så att den får längden noll.
	s = s[:0]
	printSlice(s)

	// Utöka dess längd.
	s = s[:4]
	printSlice(s)

	// Ta bort dess två första värden.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
