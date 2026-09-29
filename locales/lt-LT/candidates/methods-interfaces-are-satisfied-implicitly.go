//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Šis metodas reiškia, kad tipas T įgyvendina sąsają I,
// tačiau mums nereikia to aiškiai deklaruoti.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
