//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci የሚባለው ተግባር የሚመልሰው
// int የሚመልስ ተግባር ነው።
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
