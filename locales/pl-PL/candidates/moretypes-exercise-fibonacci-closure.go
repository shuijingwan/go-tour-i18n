//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci jest funkcją, która zwraca
// funkcję zwracającą int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
