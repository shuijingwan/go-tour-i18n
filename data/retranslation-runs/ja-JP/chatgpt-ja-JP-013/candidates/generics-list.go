//go:build OMIT

package main

// List は、任意の型の
// 値を保持する単方向連結リストを表します。
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
