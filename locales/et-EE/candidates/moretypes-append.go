//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append töötab ka viiludega, mille väärtus on nil.
	s = append(s, 0)
	printSlice(s)

	// Viil kasvab vastavalt vajadusele.
	s = append(s, 1)
	printSlice(s)

	// Korraga saab lisada rohkem kui ühe elemendi.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
