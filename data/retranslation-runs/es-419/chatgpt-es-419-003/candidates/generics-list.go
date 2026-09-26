//go:build OMIT

package main

// List representa una lista simplemente enlazada que contiene
// valores de cualquier tipo.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
