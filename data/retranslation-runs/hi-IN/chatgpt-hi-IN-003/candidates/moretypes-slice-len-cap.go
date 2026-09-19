//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// स्लाइस को फिर से स्लाइस करके उसकी लंबाई शून्य करें।
	s = s[:0]
	printSlice(s)

	// उसकी लंबाई बढ़ाएँ।
	s = s[:4]
	printSlice(s)

	// उसकी पहली दो वैल्यू हटा दें।
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
