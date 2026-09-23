//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // may type na Vertex
	v2 = Vertex{X: 1}  // implicit ang Y:0
	v3 = Vertex{}      // X:0 at Y:0
	p  = &Vertex{1, 2} // may type na *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
