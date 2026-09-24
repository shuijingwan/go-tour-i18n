//go:build OMIT

package main

import "fmt"

// Az Index visszaadja x indexét s-ben, vagy -1-et, ha nem található.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v és x típusa T, amelyre a comparable
		// típuskorlát vonatkozik, ezért itt használhatjuk a == operátort.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Az Index int értékek szeletével működik
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Az Index karakterláncok szeletével is működik
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
