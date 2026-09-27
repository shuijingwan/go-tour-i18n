//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // osoita muuttujaan i
	fmt.Println(*p) // lue i osoittimen kautta
	*p = 21         // aseta i:n arvo osoittimen kautta
	fmt.Println(i)  // tarkastele i:n uutta arvoa

	p = &j         // osoita muuttujaan j
	*p = *p / 37   // jaa j osoittimen kautta
	fmt.Println(j) // tarkastele j:n uutta arvoa
}
