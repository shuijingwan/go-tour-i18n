//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci är en funktion som returnerar
// en funktion som returnerar en int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
