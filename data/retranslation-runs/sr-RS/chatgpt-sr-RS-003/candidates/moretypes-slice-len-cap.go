//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Издвојите исечак тако да му дужина буде нула.
	s = s[:0]
	printSlice(s)

	// Проширите његову дужину.
	s = s[:4]
	printSlice(s)

	// Одбаците његове прве две вредности.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
