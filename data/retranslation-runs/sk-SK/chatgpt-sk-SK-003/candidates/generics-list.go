//go:build OMIT

package main

// List predstavuje jednosmerne prepojený zoznam, ktorý uchováva
// hodnoty ľubovoľného typu.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
