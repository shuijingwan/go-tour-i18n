//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci एक फ़ंक्शन है जो लौटाता है
// एक ऐसा फ़ंक्शन जो int लौटाता है।
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
