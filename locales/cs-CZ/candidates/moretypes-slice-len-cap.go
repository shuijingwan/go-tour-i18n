//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Vytvořte z řezu řez nulové délky.
	s = s[:0]
	printSlice(s)

	// Prodlužte jeho délku.
	s = s[:4]
	printSlice(s)

	// Odstraňte jeho první dvě hodnoty.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
