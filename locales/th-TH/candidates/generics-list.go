//go:build OMIT

package main

// List แทนลิงก์ลิสต์แบบทางเดียวที่เก็บ
// ค่าของชนิดใดก็ได้
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
