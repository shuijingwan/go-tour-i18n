//go:build OMIT

package main

// List predstavlja jednostruko povezanu listu koja sadržava
// vrijednosti bilo kojeg tipa.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
