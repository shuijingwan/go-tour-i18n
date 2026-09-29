//go:build OMIT

package main

import "fmt"

// Index atgriež x indeksu šķēlumā s vai -1, ja tas nav atrasts.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v un x tips ir T, kam ir comparable
		// ierobežojums, tāpēc šeit varam izmantot ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index darbojas ar int šķēlumu
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index darbojas arī ar string šķēlumu
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
