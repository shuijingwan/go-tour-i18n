//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append funguje i s řezy, jejichž hodnota je nil.
	s = append(s, 0)
	printSlice(s)

	// Řez se podle potřeby zvětšuje.
	s = append(s, 1)
	printSlice(s)

	// Můžeme přidat více než jeden prvek najednou.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
