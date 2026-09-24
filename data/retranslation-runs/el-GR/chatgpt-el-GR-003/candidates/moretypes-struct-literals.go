//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // έχει τύπο Vertex
	v2 = Vertex{X: 1}  // το Y:0 υπονοείται
	v3 = Vertex{}      // X:0 και Y:0
	p  = &Vertex{1, 2} // έχει τύπο *Vertex
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
