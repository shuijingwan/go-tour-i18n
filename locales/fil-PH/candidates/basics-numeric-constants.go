//go:build OMIT

package main

import "fmt"

const (
	// Gumawa ng napakalaking numero sa pamamagitan ng paglipat ng bit na 1 nang 100 puwesto pakaliwa.
	// Sa madaling salita, ang binary na numerong 1 na sinusundan ng 100 zero.
	Big = 1 << 100
	// Ilipat itong muli nang 99 na puwesto pakanan upang makuha ang 1<<1, o 2.
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
