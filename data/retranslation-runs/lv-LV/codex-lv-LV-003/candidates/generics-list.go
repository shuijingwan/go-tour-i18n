//go:build OMIT

package main

// List attēlo vienvirziena saistīto sarakstu, kas glabā
// jebkura tipa vērtības.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
