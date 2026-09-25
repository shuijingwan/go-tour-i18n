//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// स्लाइसची लांबी शून्य होईल अशी स्लाइस करा.
	s = s[:0]
	printSlice(s)

	// तिची लांबी वाढवा.
	s = s[:4]
	printSlice(s)

	// तिची पहिली दोन मूल्ये काढून टाका.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
