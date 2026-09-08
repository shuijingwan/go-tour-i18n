//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // aponta para i
	fmt.Println(*p) // lê i por meio do ponteiro
	*p = 21         // altera i por meio do ponteiro
	fmt.Println(i)  // mostra o novo valor de i

	p = &j         // aponta para j
	*p = *p / 37   // divide j por meio do ponteiro
	fmt.Println(j) // mostra o novo valor de j
}
