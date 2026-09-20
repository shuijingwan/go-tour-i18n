//go:build OMIT

package main

// List ایک singly-linked list کی نمائندگی کرتی ہے جو
// کسی بھی ٹائپ کی قدریں رکھتی ہے۔
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
