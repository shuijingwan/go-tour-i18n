//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci este o funcție care returnează
// o funcție care returnează un int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
