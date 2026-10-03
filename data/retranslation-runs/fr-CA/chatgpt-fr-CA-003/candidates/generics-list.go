//go:build OMIT

package main

// List représente une liste simplement chaînée qui contient
// des valeurs de n’importe quel type.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
