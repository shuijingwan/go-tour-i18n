//go:build OMIT

package main

// List представляє однозв’язний список, що містить
// значення будь-якого типу.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
