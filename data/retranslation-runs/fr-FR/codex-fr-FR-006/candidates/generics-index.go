//go:build OMIT

package main

import "fmt"

// Index renvoie l’indice de x dans s, ou -1 si x est introuvable.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v et x sont de type T, qui possède la contrainte
		// comparable, ce qui nous permet d’utiliser == ici.
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

	// Index fonctionne également avec un slice de chaînes de caractères
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
