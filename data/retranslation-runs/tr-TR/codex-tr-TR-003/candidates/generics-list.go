//go:build OMIT

package main

// List, herhangi bir türden değerler tutan tek yönlü bağlı bir listeyi
// temsil eder.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
