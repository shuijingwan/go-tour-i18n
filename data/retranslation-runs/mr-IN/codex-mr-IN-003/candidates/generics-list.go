//go:build OMIT

package main

// List हा कोणत्याही टाइपची
// मूल्ये धारण करणारी singly-linked list दर्शवतो.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
