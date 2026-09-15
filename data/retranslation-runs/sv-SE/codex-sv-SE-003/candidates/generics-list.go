//go:build OMIT

package main

// List representerar en enkellänkad lista som innehåller
// värden av valfri typ.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
