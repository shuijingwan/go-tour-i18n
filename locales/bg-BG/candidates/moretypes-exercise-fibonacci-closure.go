//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci е функция, която връща
// функция, връщаща стойност от тип int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
