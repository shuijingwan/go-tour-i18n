//go:build OMIT

package main

// List repræsenterer en enkeltkædet liste, der indeholder
// værdier af enhver type.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
