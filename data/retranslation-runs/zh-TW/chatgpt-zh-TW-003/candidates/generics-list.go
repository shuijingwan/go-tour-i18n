//go:build OMIT

package main

// List 表示一個單向鏈結串列，可保存
// 任意型別的值。
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
