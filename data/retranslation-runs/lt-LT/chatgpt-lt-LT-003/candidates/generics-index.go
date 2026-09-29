//go:build OMIT

package main

import "fmt"

// Index grąžina x indeksą pjūvyje s arba -1, jei x nerastas.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ir x yra T tipo, kuriam taikomas comparable
		// apribojimas, todėl čia galime naudoti ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index veikia su int reikšmių pjūviu
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index taip pat veikia su string reikšmių pjūviu
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
