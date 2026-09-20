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

	a = f  // ایک MyFloat، Abser کو implement کرتا ہے
	a = &v // ایک *Vertex، Abser کو implement کرتا ہے

	// اگلی لائن میں v ایک Vertex ہے (*Vertex نہیں)
	// اور Abser کو implement نہیں کرتا۔
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
