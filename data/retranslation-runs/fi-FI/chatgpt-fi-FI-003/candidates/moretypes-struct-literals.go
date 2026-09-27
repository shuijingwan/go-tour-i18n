//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // on tyyppiä Vertex
	v2 = Vertex{X: 1}  // Y:0 on implisiittinen
	v3 = Vertex{}      // X:0 ja Y:0
	p  = &Vertex{1, 2} // on tyyppiä *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
