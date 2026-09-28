//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Táto metóda znamená, že typ T implementuje rozhranie I,
// ale nemusíme to explicitne deklarovať.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
