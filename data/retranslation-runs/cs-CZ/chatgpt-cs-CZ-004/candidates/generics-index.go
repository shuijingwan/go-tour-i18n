//go:build OMIT

package main

import "fmt"

// Index vrátí index x v s, nebo -1, pokud jej nenajde.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v a x jsou typu T, který má omezení comparable,
		// takže zde můžeme použít ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funguje s řezem hodnot int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index funguje také s řezem řetězců
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
