//go:build OMIT

package main

import "fmt"

const (
	// Crea un numero enorme spostando il bit 1 di 100 posizioni verso sinistra.
	// In altre parole, il numero binario formato da 1 seguito da 100 zeri.
	Big = 1 << 100
	// Spostalo nuovamente di 99 posizioni verso destra, ottenendo 1<<1, ovvero 2.
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
