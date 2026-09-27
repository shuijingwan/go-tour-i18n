//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Denne metoden gjør at typen T implementerer grensesnittet I,
// men vi trenger ikke å deklarere dette eksplisitt.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
