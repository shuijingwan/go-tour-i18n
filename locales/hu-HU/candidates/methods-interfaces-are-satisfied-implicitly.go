//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Ez a metódus azt jelenti, hogy a T típus megvalósítja az I interfészt,
// de ezt nem kell külön deklarálnunk.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
