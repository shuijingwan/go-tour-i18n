//go:build OMIT

package main

import "fmt"

// Index возвращает индекс x в s или -1, если x не найден.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v и x имеют тип T, на который наложено ограничение comparable,
		// поэтому здесь можно использовать ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index работает со срезом int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index также работает со срезом string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
