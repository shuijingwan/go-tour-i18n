//go:build OMIT

package main

// List ਇੱਕ singly-linked list ਨੂੰ ਦਰਸਾਉਂਦੀ ਹੈ ਜੋ
// ਕਿਸੇ ਵੀ ਟਾਈਪ ਦੇ ਮੁੱਲ ਰੱਖਦੀ ਹੈ।
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
