//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Aquest mètode vol dir que el tipus T implementa la interfície I,
// però no cal declarar explícitament que ho fa.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
