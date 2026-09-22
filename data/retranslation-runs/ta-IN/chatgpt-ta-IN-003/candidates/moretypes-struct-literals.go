//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex வகையைக் கொண்டுள்ளது
	v2 = Vertex{X: 1}  // Y:0 மறைமுகமாக உள்ளது
	v3 = Vertex{}      // X:0 மற்றும் Y:0
	p  = &Vertex{1, 2} // *Vertex வகையைக் கொண்டுள்ளது
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
