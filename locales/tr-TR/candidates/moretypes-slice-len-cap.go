//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Uzunluğunu sıfır yapmak için dilimi dilimleyin.
	s = s[:0]
	printSlice(s)

	// Uzunluğunu artırın.
	s = s[:4]
	printSlice(s)

	// İlk iki değerini atın.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
