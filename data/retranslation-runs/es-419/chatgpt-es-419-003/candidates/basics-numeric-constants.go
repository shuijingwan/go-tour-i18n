//go:build OMIT

package main

import "fmt"

const (
	// Desplaza un bit con valor 1 cien posiciones a la izquierda para crear un número enorme.
	// Dicho de otro modo, es el número binario formado por un 1 seguido de 100 ceros.
	Big = 1 << 100
	// Desplázalo de nuevo 99 posiciones a la derecha; así obtenemos 1<<1, es decir, 2.
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
