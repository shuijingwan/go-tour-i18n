//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// See meetod tähendab, et tüüp T teostab liidese I,
// kuid seda pole vaja selgesõnaliselt deklareerida.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
