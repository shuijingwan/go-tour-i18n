//go:build OMIT

package main

import "fmt"

const (
	// Vytvoříme obrovské číslo posunutím bitu 1 doleva o 100 pozic.
	// Jinými slovy, jde o binární číslo, ve kterém za 1 následuje 100 nul.
	Big = 1 << 100
	// Posuneme ho znovu doprava o 99 pozic, takže dostaneme 1<<1, tedy 2.
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
