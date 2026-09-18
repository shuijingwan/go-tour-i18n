//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // có kiểu Vertex
	v2 = Vertex{X: 1}  // Y:0 được ngầm định
	v3 = Vertex{}      // X:0 và Y:0
	p  = &Vertex{1, 2} // có kiểu *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
