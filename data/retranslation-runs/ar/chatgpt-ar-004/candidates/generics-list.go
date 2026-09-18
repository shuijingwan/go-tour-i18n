//go:build OMIT

package main

// تمثل List قائمة أحادية الربط تحتفظ
// بقيم من أي نوع.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
