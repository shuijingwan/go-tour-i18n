//go:build OMIT

package main

import "fmt"

// Index returnerar indexet för x i s, eller -1 om det inte hittas.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v och x är av typen T, som har comparable som
		// typbegränsning, så vi kan använda == här.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index fungerar för en slice med heltal
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index fungerar även för en slice med strängar
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
