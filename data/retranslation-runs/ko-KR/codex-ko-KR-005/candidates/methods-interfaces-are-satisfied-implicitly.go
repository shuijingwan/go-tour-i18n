//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// 이 메서드는 타입 T가 인터페이스 I를 구현한다는 뜻이지만,
// 이를 명시적으로 선언할 필요는 없습니다.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
