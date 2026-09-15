//go:build OMIT

package main

import "fmt"

// Index zwraca indeks x w s albo -1, jeśli go nie znaleziono.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v i x mają typ T, który spełnia ograniczenie typu comparable,
		// więc można tutaj użyć ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index działa na wycinku wartości int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index działa również na wycinku ciągów znaków
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
