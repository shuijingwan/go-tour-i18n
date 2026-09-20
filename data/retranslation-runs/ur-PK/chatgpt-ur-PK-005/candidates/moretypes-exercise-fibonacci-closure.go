//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci ایک ایسا فنکشن ہے جو ایک فنکشن واپس کرتا ہے،
// اور واپس کیا گیا فنکشن int واپس کرتا ہے۔
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
