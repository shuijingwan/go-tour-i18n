//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci एक ऐसा फ़ंक्शन है जो
// एक ऐसा फ़ंक्शन लौटाता है जो int लौटाता है।
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
