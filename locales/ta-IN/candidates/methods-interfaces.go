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

	a = f  // MyFloat ஒன்று Abser-ஐ நடைமுறைப்படுத்துகிறது
	a = &v // *Vertex ஒன்று Abser-ஐ நடைமுறைப்படுத்துகிறது

	// அடுத்த வரியில், v என்பது Vertex (*Vertex அல்ல)
	// மேலும் Abser-ஐ நடைமுறைப்படுத்தாது.
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
