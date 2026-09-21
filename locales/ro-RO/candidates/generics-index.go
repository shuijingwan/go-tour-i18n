//go:build OMIT

package main

import "fmt"

// Index returnează indicele lui x în s sau -1 dacă nu este găsit.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v și x sunt de tip T, care are constrângerea comparable,
		// așa că putem folosi == aici.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funcționează pe un slice de int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index funcționează și pe un slice de string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
