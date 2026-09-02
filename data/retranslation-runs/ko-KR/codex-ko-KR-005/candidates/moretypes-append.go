//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append는 nil 슬라이스에서도 동작합니다.
	s = append(s, 0)
	printSlice(s)

	// 슬라이스는 필요에 따라 늘어납니다.
	s = append(s, 1)
	printSlice(s)

	// 한 번에 둘 이상의 요소를 추가할 수 있습니다.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
