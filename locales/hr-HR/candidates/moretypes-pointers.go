//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // pokazuje na i
	fmt.Println(*p) // čita i putem pokazivača
	*p = 21         // postavlja i putem pokazivača
	fmt.Println(i)  // prikazuje novu vrijednost i

	p = &j         // pokazuje na j
	*p = *p / 37   // dijeli j putem pokazivača
	fmt.Println(j) // prikazuje novu vrijednost j
}
