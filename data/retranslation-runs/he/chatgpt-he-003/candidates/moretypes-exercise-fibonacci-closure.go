//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci היא פונקציה שמחזירה
// פונקציה שמחזירה int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
