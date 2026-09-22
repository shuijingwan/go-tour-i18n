//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci అనేది int ను రిటర్న్ చేసే
// ఫంక్షన్‌ను రిటర్న్ చేసే ఫంక్షన్.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
