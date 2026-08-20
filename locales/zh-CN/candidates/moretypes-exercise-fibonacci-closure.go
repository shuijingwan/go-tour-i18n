//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci 是一个函数，它返回
// 另一个返回 int 的函数。
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
