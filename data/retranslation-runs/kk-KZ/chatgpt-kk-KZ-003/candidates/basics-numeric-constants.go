//go:build OMIT

package main

import "fmt"

const (
	// 1 битін 100 орынға солға жылжыту арқылы өте үлкен сан жасаңыз.
	// Басқаша айтқанда, бұл — 1 санынан кейін 100 нөл келетін екілік сан.
	Big = 1 << 100
	// Оны қайтадан 99 орынға оңға жылжытыңыз, сонда 1<<1, яғни 2 шығады.
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
