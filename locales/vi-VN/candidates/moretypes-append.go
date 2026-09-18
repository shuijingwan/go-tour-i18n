//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append hoạt động với slice nil.
	s = append(s, 0)
	printSlice(s)

	// Slice tăng kích thước khi cần.
	s = append(s, 1)
	printSlice(s)

	// Ta có thể thêm nhiều phần tử cùng lúc.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
