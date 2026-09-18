//go:build OMIT

package main

// List biểu diễn một danh sách liên kết đơn chứa
// các giá trị thuộc bất kỳ kiểu nào.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
