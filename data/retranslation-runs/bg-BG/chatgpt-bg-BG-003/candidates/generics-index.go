//go:build OMIT

package main

import "fmt"

// Index връща индекса на x в s или -1, ако x не е намерен.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v и x са от тип T с ограничението comparable
		// което позволява тук да използваме ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index работи със срез от цели числа
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index работи и със срез от низове
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
