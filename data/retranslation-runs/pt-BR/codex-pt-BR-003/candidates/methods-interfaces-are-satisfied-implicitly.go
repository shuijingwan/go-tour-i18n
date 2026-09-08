//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Este método indica que o tipo T implementa a interface I,
// mas não precisamos declarar isso explicitamente.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
