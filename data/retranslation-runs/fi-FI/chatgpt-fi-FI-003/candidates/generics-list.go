//go:build OMIT

package main

// List edustaa yksisuuntaisesti linkitettyä listaa, joka sisältää
// minkä tahansa tyyppisiä arvoja.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
