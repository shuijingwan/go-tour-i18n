//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci 是一個函式，它會回傳
// 另一個回傳 int 的函式。
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
