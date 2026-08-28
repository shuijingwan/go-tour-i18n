//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // hat den Typ Vertex
	v2 = Vertex{X: 1}  // Y:0 ist implizit
	v3 = Vertex{}      // X:0 und Y:0
	p  = &Vertex{1, 2} // hat den Typ *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
