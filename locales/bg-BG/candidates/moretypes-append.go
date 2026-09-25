//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append работи и със срезове със стойност nil.
	s = append(s, 0)
	printSlice(s)

	// Срезът се разширява според необходимостта.
	s = append(s, 1)
	printSlice(s)

	// Можем да добавяме повече от един елемент наведнъж.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
