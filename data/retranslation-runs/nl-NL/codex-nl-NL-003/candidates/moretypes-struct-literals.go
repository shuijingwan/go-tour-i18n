//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // heeft type Vertex
	v2 = Vertex{X: 1}  // Y:0 is impliciet
	v3 = Vertex{}      // X:0 en Y:0
	p  = &Vertex{1, 2} // heeft type *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
