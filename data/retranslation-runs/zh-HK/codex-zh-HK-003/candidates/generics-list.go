//go:build OMIT

package main

// List 代表一個單向鏈結串列，可保存
// 任何類型的值。
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
