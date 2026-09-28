//go:build OMIT

package main

import "fmt"

const (
	// Vytvor obrovské číslo posunutím bitu 1 o 100 pozícií doľava.
	// Inými slovami, binárne číslo, v ktorom po 1 nasleduje 100 núl.
	Big = 1 << 100
	// Posuň ho znova doprava o 99 pozícií, takže dostaneme 1<<1, teda 2.
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
