//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci हे असे फंक्शन आहे जे
// int परत करणारे फंक्शन परत करते.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
