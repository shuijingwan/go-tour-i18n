//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci on funktsioon, mis tagastab
// funktsiooni, mille tagastusväärtus on tüüpi int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
