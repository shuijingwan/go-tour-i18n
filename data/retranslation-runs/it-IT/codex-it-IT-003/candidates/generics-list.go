//go:build OMIT

package main

// List rappresenta una lista concatenata semplice che contiene
// valori di qualsiasi tipo.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
