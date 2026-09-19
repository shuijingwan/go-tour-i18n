//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // มีชนิดเป็น Vertex
	v2 = Vertex{X: 1}  // Y:0 ถูกกำหนดโดยปริยาย
	v3 = Vertex{}      // X:0 และ Y:0
	p  = &Vertex{1, 2} // มีชนิดเป็น *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
