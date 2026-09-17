//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Method ini berarti tipe T mengimplementasikan interface I,
// tetapi kita tidak perlu mendeklarasikannya secara eksplisit.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
