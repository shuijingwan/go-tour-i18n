//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// เมธอดนี้ทำให้ชนิด T เป็นไปตามอินเทอร์เฟซ I
// แต่เราไม่จำเป็นต้องประกาศเรื่องนี้อย่างชัดเจน
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
