//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// துண்டத்தை வெட்டி அதன் நீளத்தைச் சுழியமாக்குங்கள்.
	s = s[:0]
	printSlice(s)

	// அதன் நீளத்தை நீட்டிக்கவும்.
	s = s[:4]
	printSlice(s)

	// அதன் முதல் இரண்டு மதிப்புகளை நீக்கவும்.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
