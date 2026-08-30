//go:build OMIT

package main

import "fmt"

// Index renvoie l’indice de x dans s, ou -1 si x n’y figure pas.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v et x sont de type T, qui satisfait la contrainte
		// comparable ; nous pouvons donc utiliser == ici.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index fonctionne avec un slice d’int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index fonctionne aussi avec un slice de string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
