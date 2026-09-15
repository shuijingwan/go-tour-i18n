//go:build OMIT

package main

// List reprezentuje listę jednokierunkową, która przechowuje
// wartości dowolnego typu.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
