//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Kaedah ini bermaksud jenis T melaksanakan antara muka I,
// tetapi kita tidak perlu mengisytiharkannya secara eksplisit.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
