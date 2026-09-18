//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci là một hàm trả về
// một hàm trả về int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
