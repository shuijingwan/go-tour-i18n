//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex 型
	v2 = Vertex{X: 1}  // Y:0 は暗黙に指定されます
	v3 = Vertex{}      // X:0 と Y:0
	p  = &Vertex{1, 2} // *Vertex 型
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
