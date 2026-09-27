//go:build OMIT

package main

import "fmt"

// Index returnerer indeksen til x i s, eller -1 hvis x ikke finnes.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v og x har typen T, som har typebegrensningen comparable
		// slik at vi kan bruke == her.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index fungerer også på en slice med heltall
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index fungerer også på en slice med strenger
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
