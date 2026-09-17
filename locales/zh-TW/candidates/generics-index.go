//go:build OMIT

package main

import "fmt"

// Index 會回傳 x 在 s 中的索引；若找不到則回傳 -1。
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v 和 x 都是型別 T，而 T 具有 comparable
		// 型別約束，因此這裡可以使用 ==。
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index 適用於 int 切片
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index 也適用於 string 切片
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
