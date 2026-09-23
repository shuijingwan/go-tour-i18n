//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append berfungsi pada hirisan nil.
	s = append(s, 0)
	printSlice(s)

	// Hirisan berkembang mengikut keperluan.
	s = append(s, 1)
	printSlice(s)

	// Kita boleh menambah lebih daripada satu elemen pada satu-satu masa.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
