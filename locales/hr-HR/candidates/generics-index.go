//go:build OMIT

package main

import "fmt"

// Index vraća indeks vrijednosti x u s ili -1 ako nije pronađena.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v i x jesu vrijednosti tipa T, za koji vrijedi ograničenje comparable
		// pa se ovdje možemo koristiti operatorom ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index radi na isječku cijelih brojeva
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index radi i na isječku znakovnih nizova
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
