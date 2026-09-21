//go:build OMIT

package main

// List reprezintă o listă simplu înlănțuită care conține
// valori de orice tip.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
