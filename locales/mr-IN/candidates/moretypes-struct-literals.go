//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // याचा टाइप Vertex आहे
	v2 = Vertex{X: 1}  // Y:0 गृहीत धरले जाते
	v3 = Vertex{}      // X:0 आणि Y:0
	p  = &Vertex{1, 2} // याचा टाइप *Vertex आहे
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
