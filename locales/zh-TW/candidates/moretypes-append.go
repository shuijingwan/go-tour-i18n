//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append 可用於 nil 切片。
	s = append(s, 0)
	printSlice(s)

	// 切片會視需要擴充。
	s = append(s, 1)
	printSlice(s)

	// 我們可以一次加入多個元素。
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
