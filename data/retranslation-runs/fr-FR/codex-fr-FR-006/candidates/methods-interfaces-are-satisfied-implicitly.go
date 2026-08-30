//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Cette méthode signifie que le type T implémente l’interface I,
// mais il n’est pas nécessaire de le déclarer explicitement.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
