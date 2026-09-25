//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex ടൈപ്പാണ്
	v2 = Vertex{X: 1}  // Y:0 സ്വതവേ ലഭിക്കുന്നു
	v3 = Vertex{}      // X:0, Y:0
	p  = &Vertex{1, 2} // *Vertex ടൈപ്പാണ്
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
