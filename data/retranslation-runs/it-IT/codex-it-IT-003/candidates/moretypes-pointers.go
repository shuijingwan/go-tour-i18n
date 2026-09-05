//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // punta a i
	fmt.Println(*p) // legge i tramite il puntatore
	*p = 21         // imposta i tramite il puntatore
	fmt.Println(i)  // mostra il nuovo valore di i

	p = &j         // punta a j
	*p = *p / 37   // divide j tramite il puntatore
	fmt.Println(j) // mostra il nuovo valore di j
}
