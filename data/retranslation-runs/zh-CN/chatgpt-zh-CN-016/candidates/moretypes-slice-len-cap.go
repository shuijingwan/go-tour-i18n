//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// 将该切片重新切片，使其长度为 0。
	s = s[:0]
	printSlice(s)

	// 扩展它的长度。
	s = s[:4]
	printSlice(s)

	// 去掉前两个元素。
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
