//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // از نوع Vertex است
	v2 = Vertex{X: 1}  // Y:0 ضمنی است
	v3 = Vertex{}      // X:0 و Y:0
	p  = &Vertex{1, 2} // از نوع *Vertex است
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
