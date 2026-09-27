//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci on funktio, joka palauttaa
// int-arvon palauttavan funktion.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
