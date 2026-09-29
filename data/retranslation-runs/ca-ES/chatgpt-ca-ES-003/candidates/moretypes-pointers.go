//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // apunta a i
	fmt.Println(*p) // llegeix i mitjançant el punter
	*p = 21         // assigna i mitjançant el punter
	fmt.Println(i)  // observa el nou valor d’i

	p = &j         // apunta a j
	*p = *p / 37   // divideix j mitjançant el punter
	fmt.Println(j) // observa el nou valor de j
}
