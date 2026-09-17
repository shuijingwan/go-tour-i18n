//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append dapat digunakan pada slice nil.
	s = append(s, 0)
	printSlice(s)

	// Slice akan bertambah sesuai kebutuhan.
	s = append(s, 1)
	printSlice(s)

	// Kita dapat menambahkan lebih dari satu elemen sekaligus.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
