//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append fungerer også på nil-slice-er.
	s = append(s, 0)
	printSlice(s)

	// Slicen vokser ved behov.
	s = append(s, 1)
	printSlice(s)

	// Vi kan legge til flere elementer samtidig.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
