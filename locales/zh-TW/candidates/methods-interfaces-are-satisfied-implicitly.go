//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// 這個方法表示型別 T 實作了介面 I，
// 但我們不需要明確宣告它有這麼做。
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
