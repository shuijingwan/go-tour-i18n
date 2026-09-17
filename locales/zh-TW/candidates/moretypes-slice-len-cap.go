//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// 重新切片，讓切片長度變成零。
	s = s[:0]
	printSlice(s)

	// 延長它的長度。
	s = s[:4]
	printSlice(s)

	// 移除前兩個值。
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
