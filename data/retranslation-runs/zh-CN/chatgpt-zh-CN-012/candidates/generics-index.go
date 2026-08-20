//go:build OMIT

package main

import "fmt"

// Index 返回 x 在 s 中的索引；如果未找到，则返回 -1。
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v 和 x 的类型都是 T，T 具有 comparable
		// 约束，因此这里可以使用 ==。
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index 可用于 int 类型的切片
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index 也可用于 string 类型的切片
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
