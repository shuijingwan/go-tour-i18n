//go:build OMIT

package main

import "fmt"

const (
	// 1 değerindeki bir biti 100 bit konumu kadar sola kaydırarak çok büyük bir sayı oluşturun.
	// Başka bir deyişle bu, 1'in ardından 100 sıfır gelen ikili sayıdır.
	Big = 1 << 100
	// Onu yeniden 99 bit konumu kadar sağa kaydırın; böylece 1<<1, yani 2 elde ederiz.
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
