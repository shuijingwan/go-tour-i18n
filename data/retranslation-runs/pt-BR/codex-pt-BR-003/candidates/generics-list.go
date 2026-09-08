//go:build OMIT

package main

// List representa uma lista simplesmente encadeada que armazena
// valores de qualquer tipo.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
