//go:build OMIT

package main

import "fmt"

const (
	// با شیفت دادن بیت 1 به‌اندازهٔ 100 مکان به چپ، عددی بسیار بزرگ بسازید.
	// به‌عبارت دیگر، عدد دودویی‌ای که از 1 و سپس 100 صفر تشکیل شده است.
	Big = 1 << 100
	// آن را دوباره 99 مکان به راست شیفت دهید تا در نهایت به 1<<1، یعنی 2، برسیم.
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
