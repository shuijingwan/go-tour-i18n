//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append በ nil ቁራጮች ላይ ይሰራል።
	s = append(s, 0)
	printSlice(s)

	// ቁራጩ እንደ አስፈላጊነቱ ያድጋል።
	s = append(s, 1)
	printSlice(s)

	// በአንድ ጊዜ ከአንድ በላይ አባሎችን መጨመር እንችላለን።
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
