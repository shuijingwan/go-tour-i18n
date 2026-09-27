//go:build OMIT

package main

import "fmt"

const (
	// Dan et meget stort tal ved at forskyde en bit med værdien 1 100 pladser mod venstre.
	// Med andre ord: et binært tal, der består af et ettal efterfulgt af 100 nuller.
	Big = 1 << 100
	// Forskyd det derefter 99 pladser mod højre, så vi ender med 1<<1, altså 2.
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
