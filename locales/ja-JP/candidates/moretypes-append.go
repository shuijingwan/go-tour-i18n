//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append は nil スライスでも動作します。
	s = append(s, 0)
	printSlice(s)

	// スライスは必要に応じて拡張されます。
	s = append(s, 1)
	printSlice(s)

	// 一度に複数の要素を追加できます。
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
