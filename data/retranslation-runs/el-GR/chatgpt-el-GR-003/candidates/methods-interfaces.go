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

	a = f  // μια τιμή MyFloat υλοποιεί τη διεπαφή Abser
	a = &v // ένας *Vertex υλοποιεί τη διεπαφή Abser

	// Στην ακόλουθη γραμμή, η v είναι μια τιμή Vertex (όχι *Vertex)
	// και ΔΕΝ υλοποιεί τη διεπαφή Abser.
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
