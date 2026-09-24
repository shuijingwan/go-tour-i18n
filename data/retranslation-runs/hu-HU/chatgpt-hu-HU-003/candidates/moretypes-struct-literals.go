//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // típusa Vertex
	v2 = Vertex{X: 1}  // az Y:0 implicit
	v3 = Vertex{}      // X:0 és Y:0
	p  = &Vertex{1, 2} // típusa *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
