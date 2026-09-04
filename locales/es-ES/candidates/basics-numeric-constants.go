//go:build OMIT

package main

import "fmt"

const (
	// Crea un número enorme desplazando un bit 1 100 posiciones a la izquierda.
	// En otras palabras, el número binario formado por un 1 seguido de 100 ceros.
	Big = 1 << 100
	// Vuelve a desplazarlo 99 posiciones a la derecha, con lo que obtenemos 1<<1, es decir, 2.
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
