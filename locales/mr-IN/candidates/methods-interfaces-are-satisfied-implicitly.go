//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// या मेथडचा अर्थ T टाइप I इंटरफेस लागू करतो,
// पण तसे स्पष्टपणे घोषित करण्याची गरज नाही.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
