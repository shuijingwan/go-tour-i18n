//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // 類型為 Vertex
	v2 = Vertex{X: 1}  // 隱含 Y:0
	v3 = Vertex{}      // X:0 和 Y:0
	p  = &Vertex{1, 2} // 類型為 *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
