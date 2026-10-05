//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // auf i zeigen
	fmt.Println(*p) // i über den Zeiger lesen
	*p = 21         // i über den Zeiger setzen
	fmt.Println(i)  // den neuen Wert von i anzeigen

	p = &j         // auf j zeigen
	*p = *p / 37   // j über den Zeiger dividieren
	fmt.Println(j) // den neuen Wert von j anzeigen
}
