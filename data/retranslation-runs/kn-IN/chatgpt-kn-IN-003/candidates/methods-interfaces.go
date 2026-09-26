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

	a = f  // MyFloat ಮೌಲ್ಯವು Abser ಅನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುತ್ತದೆ
	a = &v // *Vertex ಪಾಯಿಂಟರ್ ಟೈಪ್ ಕೂಡ Abser ಅನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುತ್ತದೆ

	// ಮುಂದಿನ ಸಾಲಿನಲ್ಲಿ v ಎಂಬುದು Vertex ಮೌಲ್ಯ (*Vertex ಅಲ್ಲ);
	// ಆದ್ದರಿಂದ ಅದು Abser ಅನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುವುದಿಲ್ಲ.
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
