//go:build OMIT

package main

import "fmt"

// Index retourne l’indice de x dans s, ou -1 si x n’est pas trouvé.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v et x sont de type T, qui satisfait la
		// contrainte comparable, donc nous pouvons utiliser == ici.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index fonctionne sur une tranche de int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index fonctionne aussi sur une tranche de string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
