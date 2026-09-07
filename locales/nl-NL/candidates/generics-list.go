//go:build OMIT

package main

// List vertegenwoordigt een enkelvoudig gekoppelde lijst die
// waarden van elk type bevat.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
