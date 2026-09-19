//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// ตัดสไลซ์ให้มีความยาวเป็นศูนย์
	s = s[:0]
	printSlice(s)

	// เพิ่มความยาวของสไลซ์
	s = s[:4]
	printSlice(s)

	// ตัดสองค่าแรกออก
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
