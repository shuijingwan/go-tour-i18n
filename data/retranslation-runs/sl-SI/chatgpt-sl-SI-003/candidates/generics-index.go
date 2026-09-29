//go:build OMIT

package main

import "fmt"

// Index vrne indeks x v s oziroma -1, če x ni najden.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v in x sta tipa T, za katerega velja omejitev comparable,
		// zato lahko tukaj uporabimo ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index deluje na rezini vrednosti int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index deluje tudi na rezini nizov
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
