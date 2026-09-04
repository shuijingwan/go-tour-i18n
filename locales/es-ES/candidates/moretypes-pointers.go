//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // apunta a i
	fmt.Println(*p) // lee i mediante el puntero
	*p = 21         // asigna un valor a i mediante el puntero
	fmt.Println(i)  // observa el nuevo valor de i

	p = &j         // apunta a j
	*p = *p / 37   // divide j mediante el puntero
	fmt.Println(j) // observa el nuevo valor de j
}
