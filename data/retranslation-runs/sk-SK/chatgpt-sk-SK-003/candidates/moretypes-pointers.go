//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // ukazuj na i
	fmt.Println(*p) // čítaj i cez ukazovateľ
	*p = 21         // nastav i cez ukazovateľ
	fmt.Println(i)  // pozri novú hodnotu i

	p = &j         // ukazuj na j
	*p = *p / 37   // vydeľ j cez ukazovateľ
	fmt.Println(j) // pozri novú hodnotu j
}
