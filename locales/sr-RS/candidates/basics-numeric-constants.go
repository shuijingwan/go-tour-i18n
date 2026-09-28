//go:build OMIT

package main

import "fmt"

const (
	// Направите огроман број померањем бита 1 улево за 100 места.
	// Другим речима, бинарни број који се састоји од цифре 1 иза које следи 100 нула.
	Big = 1 << 100
	// Поново га померите удесно за 99 места, тако да добијемо 1<<1, односно 2.
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
