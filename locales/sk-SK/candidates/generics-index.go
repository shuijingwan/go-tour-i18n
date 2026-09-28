//go:build OMIT

package main

import "fmt"

// Index vracia index x v s alebo -1, ak sa nenájde.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v a x sú typu T, ktorý má obmedzenie comparable,
		// preto tu môžeme použiť ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funguje s rezom celých čísel.
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index funguje aj s rezom reťazcov.
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
