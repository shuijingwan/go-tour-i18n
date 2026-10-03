//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci 是一個會傳回
// 另一個傳回 int 之函數的函數。
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
