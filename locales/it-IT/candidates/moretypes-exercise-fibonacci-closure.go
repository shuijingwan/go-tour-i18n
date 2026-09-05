//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci è una funzione che restituisce
// una funzione che restituisce un int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
