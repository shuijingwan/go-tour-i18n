//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Den här metoden innebär att typen T implementerar gränssnittet I,
// men vi behöver inte uttryckligen deklarera att den gör det.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
