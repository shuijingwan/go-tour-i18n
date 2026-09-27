//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Denne metode betyder, at typen T implementerer interfacet I,
// men vi behøver ikke at deklarere det eksplicit.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
