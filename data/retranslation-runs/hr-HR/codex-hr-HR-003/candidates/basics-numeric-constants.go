//go:build OMIT

package main

import "fmt"

const (
	// Stvori ogroman broj pomicanjem bita 1 za 100 mjesta ulijevo.
	// Drugim riječima, binarni broj koji se sastoji od znamenke 1 i 100 nula iza nje.
	Big = 1 << 100
	// Ponovno ga pomakni za 99 mjesta udesno, tako da dobijemo 1<<1, odnosno 2.
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
