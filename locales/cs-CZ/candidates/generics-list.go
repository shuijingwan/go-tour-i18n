//go:build OMIT

package main

// List představuje jednosměrně vázaný seznam, který obsahuje
// hodnoty libovolného typu.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
