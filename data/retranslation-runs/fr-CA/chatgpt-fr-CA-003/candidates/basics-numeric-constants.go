//go:build OMIT

package main

import "fmt"

const (
	// Créez un nombre énorme en décalant un bit 1 de 100 positions vers la gauche.
	// Autrement dit, le nombre binaire composé d’un 1 suivi de 100 zéros.
	Big = 1 << 100
	// Décalez-le à nouveau de 99 positions vers la droite, de sorte à obtenir 1<<1, soit 2.
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
