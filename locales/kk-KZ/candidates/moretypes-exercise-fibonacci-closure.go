//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci — функция,
// ол int қайтаратын функцияны қайтарады.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
