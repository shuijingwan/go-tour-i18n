//go:build OMIT

package main

import "fmt"

const (
	// Cipta nombor yang sangat besar dengan mengalih bit 1 ke kiri sebanyak 100 tempat.
	// Dengan kata lain, nombor perduaan yang terdiri daripada 1 diikuti oleh 100 sifar.
	Big = 1 << 100
	// Alih semula ke kanan sebanyak 99 tempat, supaya hasilnya 1<<1, atau 2.
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
