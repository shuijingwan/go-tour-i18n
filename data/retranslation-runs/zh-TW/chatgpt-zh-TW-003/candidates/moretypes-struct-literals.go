//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // 型別為 Vertex
	v2 = Vertex{X: 1}  // Y:0 是隱含的
	v3 = Vertex{}      // X:0 與 Y:0
	p  = &Vertex{1, 2} // 型別為 *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
