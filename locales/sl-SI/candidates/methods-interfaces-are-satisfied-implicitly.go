//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Ta metoda pomeni, da tip T implementira vmesnik I,
// vendar nam tega ni treba izrecno deklarirati.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
