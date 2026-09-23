//go:build OMIT

package main

import "fmt"

// Ibinabalik ng Index ang index ng x sa s, o -1 kung hindi ito natagpuan.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// Ang v at x ay may type na T, na may comparable
		// constraint, kaya maaari nating gamitin ang == rito.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Gumagana ang Index sa isang slice ng mga int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Gumagana rin ang Index sa isang slice ng mga string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
