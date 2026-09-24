//go:build nobuild || OMIT

package main

import "fmt"

// Η fibonacci είναι μια συνάρτηση που επιστρέφει
// μια συνάρτηση που επιστρέφει μια τιμή int.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
