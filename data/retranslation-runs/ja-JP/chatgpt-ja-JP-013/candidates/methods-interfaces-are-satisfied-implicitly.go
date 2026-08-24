//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// このメソッドにより、型 T はインターフェース I を実装しますが、
// そのことを明示的に宣言する必要はありません。
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
