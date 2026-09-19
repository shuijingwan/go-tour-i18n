//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// इस मेथड का अर्थ है कि टाइप T इंटरफ़ेस I को लागू करता है,
// लेकिन हमें इसे स्पष्ट रूप से घोषित करने की आवश्यकता नहीं है।
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
