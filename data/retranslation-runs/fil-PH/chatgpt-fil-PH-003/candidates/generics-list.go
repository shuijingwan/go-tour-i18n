//go:build OMIT

package main

// Ang List ay kumakatawan sa isang singly-linked list na naglalaman ng
// mga value ng anumang type.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
