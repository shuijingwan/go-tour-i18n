//go:build OMIT

package main

import "fmt"

// Index returnerer indekset for x i s eller -1, hvis x ikke findes.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v og x har typen T, som har typebegrænsningen comparable,
		// så vi kan bruge == her.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index virker med en slice af int-værdier
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index virker også med en slice af strenge
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
