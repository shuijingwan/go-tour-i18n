//go:build OMIT

package main

import "fmt"

const (
	// Luo valtava luku siirtämällä ykkösbittiä 100 paikkaa vasemmalle.
	// Toisin sanoen tuloksena on binääriluku, jossa ykköstä seuraa 100 nollaa.
	Big = 1 << 100
	// Siirrä lukua sitten 99 paikkaa takaisin oikealle, jolloin tulokseksi saadaan 1<<1 eli 2.
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
