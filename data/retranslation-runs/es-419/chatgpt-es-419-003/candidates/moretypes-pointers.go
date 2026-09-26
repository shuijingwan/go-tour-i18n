//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // apuntar a i
	fmt.Println(*p) // leer i mediante el puntero
	*p = 21         // asignar un valor a i mediante el puntero
	fmt.Println(i)  // ver el nuevo valor de i

	p = &j         // apuntar a j
	*p = *p / 37   // dividir j mediante el puntero
	fmt.Println(j) // ver el nuevo valor de j
}
