//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci는 int를 반환하는 함수를
// 반환하는 함수입니다.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
