//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // اس کی ٹائپ Vertex ہے
	v2 = Vertex{X: 1}  // Y:0 implicit ہے
	v3 = Vertex{}      // X:0 اور Y:0
	p  = &Vertex{1, 2} // اس کی ٹائپ *Vertex ہے
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
