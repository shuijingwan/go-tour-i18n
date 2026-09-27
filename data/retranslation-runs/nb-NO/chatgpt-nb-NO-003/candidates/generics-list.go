//go:build OMIT

package main

// List representerer en enkeltlenket liste som inneholder
// verdier av en hvilken som helst type.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
