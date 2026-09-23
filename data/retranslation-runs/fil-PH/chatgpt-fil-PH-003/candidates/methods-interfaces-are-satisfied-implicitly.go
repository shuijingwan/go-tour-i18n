//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Ipinapahiwatig ng method na ito na ini-implement ng type na T ang interface na I,
// ngunit hindi natin kailangang tahasang ideklara na ginagawa nito iyon.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
