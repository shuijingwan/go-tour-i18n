//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Questo metodo implica che il tipo T implementa l'interfaccia I,
// ma non è necessario dichiararlo esplicitamente.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
