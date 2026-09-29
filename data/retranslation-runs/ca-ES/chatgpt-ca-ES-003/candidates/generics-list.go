//go:build OMIT

package main

// List representa una llista simplement enllaçada que conté
// valors de qualsevol tipus.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
