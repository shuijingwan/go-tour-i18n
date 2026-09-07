//go:build OMIT

package main

import "fmt"

const (
	// Maak een enorm getal door een 1-bit 100 posities naar links te verschuiven.
	// Met andere woorden, het binaire getal dat bestaat uit een 1 gevolgd door 100 nullen.
	Big = 1 << 100
	// Verschuif het weer 99 posities naar rechts, zodat we uitkomen op 1<<1, oftewel 2.
	Small = Big >> 99
)

func needInt(x int) int { return x*10 + 1 }
func needFloat(x float64) float64 {
	return x * 0.1
}

func main() {
	fmt.Println(needInt(Small))
	fmt.Println(needFloat(Small))
	fmt.Println(needFloat(Big))
}
