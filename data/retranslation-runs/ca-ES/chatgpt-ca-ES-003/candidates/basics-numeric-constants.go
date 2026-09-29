//go:build OMIT

package main

import "fmt"

const (
	// Crea un nombre enorme desplaçant un bit 1 100 posicions cap a l’esquerra.
	// Dit d’una altra manera, el nombre binari que és un 1 seguit de 100 zeros.
	Big = 1 << 100
	// Torna’l a desplaçar 99 posicions cap a la dreta, de manera que acabem amb 1<<1, és a dir, 2.
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
