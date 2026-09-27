//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci er en funktion, der returnerer
// en funktion, som returnerer en int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
