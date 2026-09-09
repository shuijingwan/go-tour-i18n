//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Bu metot, T türünün I arayüzünü uyguladığı anlamına gelir,
// ancak bunu açıkça bildirmemiz gerekmez.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
