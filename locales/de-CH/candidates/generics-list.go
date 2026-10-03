//go:build OMIT

package main

// List stellt eine einfach verkettete Liste dar, die
// Werte beliebigen Typs enthält.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
