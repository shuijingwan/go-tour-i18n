//go:build OMIT

package main

import "fmt"

// Index gibt den Index von x in s zurück, oder -1, wenn x nicht gefunden wird.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v und x haben den Typ T, der den comparable
		// Constraint erfüllt, daher können wir hier == verwenden.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funktioniert mit einem Slice mit int-Werten
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index funktioniert auch mit einem Slice mit String-Werten
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
