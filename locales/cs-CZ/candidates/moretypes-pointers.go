//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // ukazatel na i
	fmt.Println(*p) // čtení i přes ukazatel
	*p = 21         // nastavení i přes ukazatel
	fmt.Println(i)  // zobrazení nové hodnoty i

	p = &j         // ukazatel na j
	*p = *p / 37   // dělení j přes ukazatel
	fmt.Println(j) // zobrazení nové hodnoty j
}
