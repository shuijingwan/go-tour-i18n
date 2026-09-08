//go:build OMIT

package main

import "fmt"

// Index retorna o índice de x em s ou -1 se não for encontrado.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v e x são do tipo T, que tem a
		// restrição comparable, portanto podemos usar == aqui.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index funciona com um slice de ints
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index também funciona com um slice de strings
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
