//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// 对切片再次切片，使其长度为零。
	s = s[:0]
	printSlice(s)

	// 扩展其长度。
	s = s[:4]
	printSlice(s)

	// 丢弃开头两个值。
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
