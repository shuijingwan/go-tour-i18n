//go:build nobuild || OMIT

package main

import (
	"fmt"
	"math"
)

type Abser interface {
	Abs() float64
}

func main() {
	var a Abser
	f := MyFloat(-math.Sqrt2)
	v := Vertex{3, 4}

	a = f  // MyFloat, Abser નું અમલીકરણ કરે છે.
	a = &v // *Vertex, Abser નું અમલીકરણ કરે છે.

	// આગળની લાઇનમાં v એ Vertex છે (*Vertex નથી)
	// અને તે Abser નું અમલીકરણ કરતું નથી.
	a = v

	fmt.Println(a.Abs())
}

type MyFloat float64

func (f MyFloat) Abs() float64 {
	if f < 0 {
		return float64(-f)
	}
	return float64(f)
}

type Vertex struct {
	X, Y float64
}

func (v *Vertex) Abs() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}
