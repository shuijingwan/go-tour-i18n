//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // ustaw wskaźnik na i
	fmt.Println(*p) // odczytaj i przez wskaźnik
	*p = 21         // ustaw i przez wskaźnik
	fmt.Println(i)  // zobacz nową wartość i

	p = &j         // ustaw wskaźnik na j
	*p = *p / 37   // podziel j, korzystając ze wskaźnika
	fmt.Println(j) // zobacz nową wartość j
}
