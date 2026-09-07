//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci is een functie die
// een functie retourneert die een int retourneert.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
