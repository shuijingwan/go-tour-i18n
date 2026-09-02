//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// 슬라이스의 길이가 0이 되도록 슬라이싱합니다.
	s = s[:0]
	printSlice(s)

	// 길이를 늘립니다.
	s = s[:4]
	printSlice(s)

	// 처음 두 값을 제거합니다.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
