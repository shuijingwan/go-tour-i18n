//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci ni kitendakazi kinachorejesha
// kitendakazi kinachorejesha int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
