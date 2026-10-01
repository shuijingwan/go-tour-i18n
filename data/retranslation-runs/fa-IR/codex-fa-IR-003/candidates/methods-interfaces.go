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

	a = f  // یک MyFloat رابط Abser را پیاده‌سازی می‌کند
	a = &v // یک *Vertex رابط Abser را پیاده‌سازی می‌کند

	// در خط بعد، v یک Vertex است (نه *Vertex)
	// و Abser را پیاده‌سازی نمی‌کند.
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
