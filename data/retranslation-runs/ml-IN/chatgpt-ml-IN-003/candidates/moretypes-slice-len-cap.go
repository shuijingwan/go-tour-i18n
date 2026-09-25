//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// സ്ലൈസിന്റെ നീളം പൂജ്യമാകുന്ന വിധം വീണ്ടും സ്ലൈസ് ചെയ്യുക.
	s = s[:0]
	printSlice(s)

	// അതിന്റെ നീളം വർധിപ്പിക്കുക.
	s = s[:4]
	printSlice(s)

	// ആദ്യ രണ്ട് മൂല്യങ്ങൾ ഒഴിവാക്കുക.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
