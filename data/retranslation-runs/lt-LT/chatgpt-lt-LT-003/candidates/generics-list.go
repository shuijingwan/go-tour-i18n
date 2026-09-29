//go:build OMIT

package main

// List vaizduoja vienkryptį susietąjį sąrašą, kuriame saugomos
// bet kokio tipo reikšmės.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
