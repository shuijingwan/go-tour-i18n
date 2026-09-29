//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci yra funkcija, kuri grąžina
// funkciją, grąžinančią int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
