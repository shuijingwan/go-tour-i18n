//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci は、
// int を返す関数を返します。
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
