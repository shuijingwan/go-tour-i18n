//go:build OMIT

package main

import "fmt"

// Index tagastab viilus s väärtuse x indeksi või -1, kui seda ei leita.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ja x on tüüpi T, millel on comparable
		// tüübipiirang, seega saame siin kasutada ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index töötab täisarvude viiluga
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index töötab ka sõnede viiluga
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
