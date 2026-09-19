//go:build OMIT

package main

import "fmt"

// Index คืนดัชนีของ x ใน s หรือ -1 หากหาไม่พบ
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v และ x เป็นชนิด T ซึ่งมีข้อจำกัดของชนิด comparable
		// จึงสามารถใช้ == ตรงนี้ได้
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index ใช้ได้กับสไลซ์ของ int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index ใช้ได้กับสไลซ์ของ string เช่นกัน
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
