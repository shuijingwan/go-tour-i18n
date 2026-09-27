//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci એવું ફંક્શન છે જે
// int પરત કરતું બીજું ફંક્શન પરત કરે છે.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
