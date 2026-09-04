//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Reduce la slice para que tenga longitud cero.
	s = s[:0]
	printSlice(s)

	// Amplía su longitud.
	s = s[:4]
	printSlice(s)

	// Descarta sus dos primeros valores.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
