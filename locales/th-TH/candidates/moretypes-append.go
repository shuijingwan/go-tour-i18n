//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append ใช้ได้กับสไลซ์ nil
	s = append(s, 0)
	printSlice(s)

	// สไลซ์จะขยายตามความจำเป็น
	s = append(s, 1)
	printSlice(s)

	// เราสามารถเพิ่มองค์ประกอบได้มากกว่าหนึ่งค่าในแต่ละครั้ง
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
