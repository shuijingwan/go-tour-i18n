//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // 타입은 Vertex입니다
	v2 = Vertex{X: 1}  // Y:0은 암시적으로 지정됩니다
	v3 = Vertex{}      // X:0과 Y:0
	p  = &Vertex{1, 2} // 타입은 *Vertex입니다
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
