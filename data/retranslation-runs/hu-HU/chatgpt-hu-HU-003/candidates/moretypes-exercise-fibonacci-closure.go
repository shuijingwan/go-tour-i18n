//go:build nobuild || OMIT

package main

import "fmt"

// A fibonacci egy függvény, amely visszaad
// egy int értéket visszaadó függvényt.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
