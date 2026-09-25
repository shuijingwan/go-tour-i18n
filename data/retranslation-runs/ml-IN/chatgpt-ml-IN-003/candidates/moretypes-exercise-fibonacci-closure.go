//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci എന്നത് തിരികെ നൽകുന്നത്
// ഒരു int തിരികെ നൽകുന്ന ഫങ്ഷനെയാണ്.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
