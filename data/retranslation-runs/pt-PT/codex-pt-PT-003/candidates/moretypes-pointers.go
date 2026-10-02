//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // apontar para i
	fmt.Println(*p) // ler i através do ponteiro
	*p = 21         // definir i através do ponteiro
	fmt.Println(i)  // ver o novo valor de i

	p = &j         // apontar para j
	*p = *p / 37   // dividir j através do ponteiro
	fmt.Println(j) // ver o novo valor de j
}
