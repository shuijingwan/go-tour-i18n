//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// המתודה הזו גורמת לכך שהטיפוס T מממש את הממשק I,
// אבל אין צורך להצהיר במפורש שהוא עושה זאת.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
