//go:build OMIT

package main

// List 表示一个单向链表，用于保存
// 任意类型的值。
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
