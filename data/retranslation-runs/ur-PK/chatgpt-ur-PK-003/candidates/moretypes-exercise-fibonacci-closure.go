//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci ایک ایسا فنکشن ہے جو واپس کرتا ہے
// ایک فنکشن جو int واپس کرتا ہے۔
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
