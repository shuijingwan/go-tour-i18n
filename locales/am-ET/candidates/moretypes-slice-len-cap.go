//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// ቁራጩን ቁረጥና ርዝመቱን ዜሮ አድርግ።
	s = s[:0]
	printSlice(s)

	// ርዝመቱን አራዝም።
	s = s[:4]
	printSlice(s)

	// የመጀመሪያዎቹን ሁለት እሴቶች አስወግድ።
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
