//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // pointe vers i
	fmt.Println(*p) // lire i par l’intermédiaire du pointeur
	*p = 21         // modifier i par l’intermédiaire du pointeur
	fmt.Println(i)  // voir la nouvelle valeur de i

	p = &j         // pointe vers j
	*p = *p / 37   // diviser j par l’intermédiaire du pointeur
	fmt.Println(j) // voir la nouvelle valeur de j
}
