//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Ова метода значи да тип T имплементира интерфејс I,
// али не морамо експлицитно да декларишемо да то чини.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
