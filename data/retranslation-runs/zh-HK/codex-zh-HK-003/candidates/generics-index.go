//go:build OMIT

package main

import "fmt"

// Index 會傳回 x 在 s 中的索引；如果找不到，則傳回 -1。
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v 和 x 都屬於類型 T，而 T 具有 comparable
		// 約束，因此我們可以在這裡使用 ==。
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

	// Index 也適用於字串切片
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
