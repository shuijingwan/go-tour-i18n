//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci é uma função que retorna
// uma função que retorna um int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
