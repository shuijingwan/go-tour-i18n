//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// 此方法意味着类型 T 实现了接口 I，
// 但我们无需显式声明这一点。
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
