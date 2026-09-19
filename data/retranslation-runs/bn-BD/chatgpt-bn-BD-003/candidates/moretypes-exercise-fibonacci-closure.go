//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci এমন একটি ফাংশন, যা রিটার্ন করে
// এমন একটি ফাংশন, যা একটি int রিটার্ন করে।
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
