//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci, bir fonksiyon döndüren
// bir fonksiyondur; o fonksiyon da int döndürür.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
