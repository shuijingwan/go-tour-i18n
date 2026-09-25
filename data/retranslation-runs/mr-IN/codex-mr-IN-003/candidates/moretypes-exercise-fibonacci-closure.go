//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci हे असे function आहे जे परत करते
// int परत करणारे function.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
