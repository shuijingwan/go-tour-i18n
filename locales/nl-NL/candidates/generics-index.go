//go:build OMIT

package main

import "fmt"

// Index retourneert de index van x in s, of -1 als deze niet wordt gevonden.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v en x zijn van type T, waarvoor de beperking comparable geldt,
		// dus kunnen we hier == gebruiken.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index werkt op een slice van ints
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index werkt ook op een slice van strings
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
