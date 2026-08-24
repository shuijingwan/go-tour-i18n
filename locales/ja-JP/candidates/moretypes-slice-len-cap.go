//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// スライスを再スライスして長さを 0 にします。
	s = s[:0]
	printSlice(s)

	// 長さを拡張します。
	s = s[:4]
	printSlice(s)

	// 先頭の 2 つの値を取り除きます。
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
