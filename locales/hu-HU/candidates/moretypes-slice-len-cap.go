//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Szeleteld úgy a szeletet, hogy a hossza nulla legyen.
	s = s[:0]
	printSlice(s)

	// Növeld meg a hosszát.
	s = s[:4]
	printSlice(s)

	// Hagyd el az első két értékét.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
