//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci és una funció que retorna
// una funció que retorna un int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
