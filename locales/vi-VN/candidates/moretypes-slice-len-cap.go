//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Cắt slice để đưa độ dài của nó về 0.
	s = s[:0]
	printSlice(s)

	// Mở rộng độ dài của nó.
	s = s[:4]
	printSlice(s)

	// Bỏ hai giá trị đầu tiên của nó.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
