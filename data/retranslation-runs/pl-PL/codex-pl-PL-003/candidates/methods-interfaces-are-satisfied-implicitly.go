//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Ta metoda oznacza, że typ T implementuje interfejs I,
// ale nie trzeba tego jawnie deklarować.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
