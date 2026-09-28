//go:build OMIT

package main

import "fmt"

// Index враћа индекс вредности x у s или -1 ако није пронађена.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v и x су типа T, који има
		// ограничење comparable, па овде можемо да користимо ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index ради над исечком вредности типа int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index такође ради над исечком стрингова
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
