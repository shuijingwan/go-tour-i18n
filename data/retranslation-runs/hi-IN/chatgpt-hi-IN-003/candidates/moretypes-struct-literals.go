//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // इसका टाइप Vertex है
	v2 = Vertex{X: 1}  // Y:0 निहित है
	v3 = Vertex{}      // X:0 और Y:0
	p  = &Vertex{1, 2} // इसका टाइप *Vertex है
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
