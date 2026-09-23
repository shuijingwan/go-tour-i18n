//go:build nobuild || OMIT

package main

import "fmt"

// Ang fibonacci ay isang function na nagbabalik ng
// isang function na nagbabalik ng int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
