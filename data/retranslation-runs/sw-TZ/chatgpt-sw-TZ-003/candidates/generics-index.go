//go:build OMIT

package main

import "fmt"

// Index hurejesha nafasi ya x katika s, au -1 ikiwa haijapatikana.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v na x ni za aina T, ambayo ina kizuizi comparable
		// hivyo tunaweza kutumia == hapa.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index hufanya kazi kwenye kipande cha int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index pia hufanya kazi kwenye kipande cha mifuatano ya maandishi
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
