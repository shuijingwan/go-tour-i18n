//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// Gumagana ang append sa mga nil na slice.
	s = append(s, 0)
	printSlice(s)

	// Lumalaki ang slice kung kinakailangan.
	s = append(s, 1)
	printSlice(s)

	// Maaari tayong magdagdag ng higit sa isang elemento sa bawat pagkakataon.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
