//go:build OMIT

package main

// List ही एकदिश दुव्यांची यादी आहे, जी
// कोणत्याही टाइपची मूल्ये धारण करते.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
