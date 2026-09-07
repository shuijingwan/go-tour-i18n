//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Deze methode betekent dat het type T de interface I implementeert,
// maar we hoeven niet expliciet te declareren dat dit zo is.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
