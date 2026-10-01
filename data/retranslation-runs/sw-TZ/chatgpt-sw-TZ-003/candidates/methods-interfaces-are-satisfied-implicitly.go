//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Mbinu hii inamaanisha aina T inatekeleza kiolesura I,
// lakini hatuhitaji kutangaza wazi kwamba inafanya hivyo.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
