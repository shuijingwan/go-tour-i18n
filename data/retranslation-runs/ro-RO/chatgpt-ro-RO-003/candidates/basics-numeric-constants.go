//go:build OMIT

package main

import "fmt"

const (
	// Creează un număr foarte mare deplasând un bit 1 la stânga cu 100 de poziții.
	// Cu alte cuvinte, numărul binar format din 1 urmat de 100 de zerouri.
	Big = 1 << 100
	// Deplasează-l din nou la dreapta cu 99 de poziții, astfel încât să obținem 1<<1, adică 2.
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
