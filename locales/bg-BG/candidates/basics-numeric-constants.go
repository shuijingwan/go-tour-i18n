//go:build OMIT

package main

import "fmt"

const (
	// Създайте огромно число, като изместите бит със стойност 1 със 100 позиции наляво.
	// С други думи, получаваме двоичното число 1, последвано от 100 нули.
	Big = 1 << 100
	// Изместете го отново надясно с 99 позиции, така че да получим 1<<1, тоест 2.
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
