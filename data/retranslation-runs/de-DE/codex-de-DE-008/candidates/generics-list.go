//go:build OMIT

package main

// List repräsentiert eine einfach verkettete Liste, die
// Werte beliebigen Typs enthält.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
