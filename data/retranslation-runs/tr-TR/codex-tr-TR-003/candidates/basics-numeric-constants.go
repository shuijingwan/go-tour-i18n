//go:build OMIT

package main

import "fmt"

const (
	// 1 bitini 100 basamak sola kaydırarak devasa bir sayı oluşturun.
	// Başka bir deyişle, 1'in ardından 100 sıfır gelen ikili sayı.
	Big = 1 << 100
	// Onu yeniden 99 basamak sağa kaydırın; böylece 1<<1, yani 2 elde ederiz.
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
