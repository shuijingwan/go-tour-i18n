//go:build OMIT

package main

import "fmt"

// Index restituisce l'indice di x in s, oppure -1 se non viene trovato.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v e x sono di tipo T, che ha comparable come
		// vincolo, quindi qui possiamo usare ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funziona su una slice di int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index funziona anche su una slice di stringhe
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
