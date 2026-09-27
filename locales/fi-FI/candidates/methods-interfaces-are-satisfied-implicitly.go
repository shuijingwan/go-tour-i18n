//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Tämä metodi tarkoittaa, että tyyppi T toteuttaa rajapinnan I,
// mutta toteutusta ei tarvitse ilmoittaa erikseen.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
