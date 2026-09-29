//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Pakeiskite pjūvio ribas taip, kad jo ilgis būtų nulis.
	s = s[:0]
	printSlice(s)

	// Padidinkite jo ilgį.
	s = s[:4]
	printSlice(s)

	// Atmeskite pirmąsias dvi jo reikšmes.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
