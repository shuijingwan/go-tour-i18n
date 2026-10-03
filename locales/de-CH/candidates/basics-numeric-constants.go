//go:build OMIT

package main

import "fmt"

const (
	// Erzeuge eine riesige Zahl, indem das 1-Bit um 100 Stellen nach links verschoben wird.
	// Mit anderen Worten: die Binärzahl, die aus einer 1 gefolgt von 100 Nullen besteht.
	Big = 1 << 100
	// Verschiebe sie wieder um 99 Stellen nach rechts, sodass 1<<1, also 2, übrig bleibt.
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
