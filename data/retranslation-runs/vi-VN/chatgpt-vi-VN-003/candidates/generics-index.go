//go:build OMIT

package main

import "fmt"

// Index trả về chỉ số của x trong s, hoặc -1 nếu không tìm thấy.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v và x có kiểu T, kiểu này có ràng buộc kiểu comparable
		// nên ta có thể dùng == ở đây.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index hoạt động với một slice các int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index cũng hoạt động với một slice các string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
