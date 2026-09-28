//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci је функција која враћа
// функцију која враћа int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
