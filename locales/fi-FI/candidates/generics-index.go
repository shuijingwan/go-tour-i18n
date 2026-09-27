//go:build OMIT

package main

import "fmt"

// Index palauttaa arvon x indeksin viipaleessa s tai -1, jos arvoa ei löydy.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ja x ovat tyyppiä T, jonka tyyppirajoite on comparable,
		// joten tässä voidaan käyttää ==-operaattoria.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index toimii int-arvojen viipaleella
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index toimii myös string-arvojen viipaleella
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
