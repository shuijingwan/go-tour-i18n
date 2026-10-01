//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex түріне ие
	v2 = Vertex{X: 1}  // Y:0 жасырын түрде беріледі
	v3 = Vertex{}      // X:0 және Y:0
	p  = &Vertex{1, 2} // *Vertex түріне ие
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
