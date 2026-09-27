//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex ટાઇપનું છે
	v2 = Vertex{X: 1}  // Y:0 ગર્ભિત રીતે નક્કી થાય છે
	v3 = Vertex{}      // X:0 અને Y:0 બંને
	p  = &Vertex{1, 2} // *Vertex ટાઇપનું છે
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
