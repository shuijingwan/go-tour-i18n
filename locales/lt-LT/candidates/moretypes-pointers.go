//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // rodyti į i
	fmt.Println(*p) // nuskaityti i per rodyklę
	*p = 21         // nustatyti i per rodyklę
	fmt.Println(i)  // pamatyti naują i reikšmę

	p = &j         // rodyti į j
	*p = *p / 37   // padalyti j per rodyklę
	fmt.Println(j) // pamatyti naują j reikšmę
}
