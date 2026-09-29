//go:build OMIT

package main

// List predstavlja enosmerno povezani seznam, ki hrani
// vrednosti poljubnega tipa.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
