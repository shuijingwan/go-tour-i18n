//go:build OMIT

package main

import "fmt"

// Index, s-ൽ x ഉള്ള ഇൻഡെക്സ് തിരികെ നൽകുന്നു; കണ്ടെത്തിയില്ലെങ്കിൽ -1 തിരികെ നൽകുന്നു.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v, x എന്നിവ comparable
		// നിബന്ധനയുള്ള T ടൈപ്പിലാണ്; അതിനാൽ ഇവിടെ == ഉപയോഗിക്കാം.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index പൂർണസംഖ്യകളുടെ സ്ലൈസിൽ പ്രവർത്തിക്കുന്നു
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index സ്ട്രിംഗുകളുടെ സ്ലൈസിലും പ്രവർത്തിക്കുന്നു
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
