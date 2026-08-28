//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci ist eine Funktion, die
// eine Funktion zurückgibt, die einen int-Wert zurückgibt.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
