//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append हेnilस्लाइसवर काम करते.
	s = append(s, 0)
	printSlice(s)

	// गरजेनुसार स्लाइसची वाढ होते.
	s = append(s, 1)
	printSlice(s)

	// एका वेळी एकापेक्षा जास्त घटकही जोडता येतात.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
