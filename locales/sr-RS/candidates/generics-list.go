//go:build OMIT

package main

// List представља једноструко повезану листу која садржи
// вредности било ког типа.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
