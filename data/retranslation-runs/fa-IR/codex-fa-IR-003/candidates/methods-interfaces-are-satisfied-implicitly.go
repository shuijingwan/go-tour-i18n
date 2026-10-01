//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// این متد به این معناست که نوع T رابط I را پیاده‌سازی می‌کند،
// اما نیازی نیست این موضوع را به‌صراحت اعلان کنیم.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
