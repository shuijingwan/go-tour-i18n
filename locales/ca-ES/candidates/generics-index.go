//go:build OMIT

package main

import "fmt"

// Index retorna l’índex de x a s, o -1 si no es troba.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v i x són de tipus T, que té la restricció comparable,
		// per tant aquí podem utilitzar ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funciona amb un slice d’enters
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index també funciona amb un slice de cadenes
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
