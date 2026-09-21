//go:build OMIT

package main

import "fmt"

const (
	// Створимо величезне число, зсунувши біт 1 ліворуч на 100 позицій.
	// Інакше кажучи, це двійкове число, у якому після 1 ідуть 100 нулів.
	Big = 1 << 100
	// Зсунемо його знову праворуч на 99 позицій і отримаємо 1<<1, тобто 2.
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
