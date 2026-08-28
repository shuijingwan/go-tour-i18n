//go:build OMIT

package main

import "fmt"

const (
	// Erzeuge eine riesige Zahl, indem ein 1-Bit um 100 Stellen nach links verschoben wird.
	// Anders ausgedrückt: die Binärzahl, die aus einer 1 und 100 darauf folgenden Nullen besteht.
	Big = 1 << 100
	// Verschiebe sie wieder um 99 Stellen nach rechts, sodass sich 1<<1, also 2, ergibt.
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
