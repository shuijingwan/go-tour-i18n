//go:build OMIT

package main

import "fmt"

// Index በ s ውስጥ ያለውን x መረጃ ጠቋሚ ይመልሳል፤ ካልተገኘ -1 ይመልሳል።
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v እና x ዓይነታቸው T ነው፤ T ደግሞ comparable
		// ገደብ አለው፣ ስለዚህ == እዚህ መጠቀም እንችላለን።
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index በ int እሴቶች ቁራጭ ላይ ይሰራል
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index በ string እሴቶች ቁራጭ ላይም ይሰራል
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
