//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Diese Methode bedeutet, dass der Typ T das Interface I implementiert,
// wir dies jedoch nicht ausdrücklich deklarieren müssen.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
