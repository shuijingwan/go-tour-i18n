//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// يعني هذا التابع أن النوع T ينفّذ الواجهة I،
// لكننا لا نحتاج إلى التصريح بذلك صراحةً.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
