//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci est une fonction qui retourne
// une fonction qui retourne un int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
