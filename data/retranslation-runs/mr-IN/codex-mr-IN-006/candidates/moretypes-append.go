//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append हे nil स्लाइसवर काम करते.
	s = append(s, 0)
	printSlice(s)

	// स्लाइस आवश्यकतेनुसार वाढते.
	s = append(s, 1)
	printSlice(s)

	// एका वेळी एकापेक्षा जास्त घटक जोडता येतात.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
