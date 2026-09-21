//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Această metodă înseamnă că tipul T implementează interfața I,
// dar nu trebuie să declarăm explicit acest lucru.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
