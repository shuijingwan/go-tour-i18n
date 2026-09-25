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

	a = f  // ഒരു MyFloat, Abser നടപ്പാക്കുന്നു
	a = &v // ഒരു *Vertex, Abser നടപ്പാക്കുന്നു

	// താഴെയുള്ള വരിയിൽ v ഒരു Vertex ആണ് (*Vertex അല്ല)
	// അതുകൊണ്ട് അത് Abser നടപ്പാക്കുന്നില്ല.
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
