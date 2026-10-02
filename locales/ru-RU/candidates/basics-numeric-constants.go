//go:build OMIT

package main

import "fmt"

const (
	// Создадим огромное число, сдвинув бит 1 влево на 100 позиций.
	// Иными словами, это двоичное число: 1, за которой следуют 100 нулей.
	Big = 1 << 100
	// Сдвинем его вправо на 99 позиций, чтобы получить 1<<1, то есть 2.
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
