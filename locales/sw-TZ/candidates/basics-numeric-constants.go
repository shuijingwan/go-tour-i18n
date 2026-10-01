//go:build OMIT

package main

import "fmt"

const (
	// Unda nambari kubwa sana kwa kuhamisha biti ya 1 kwenda kushoto nafasi 100.
	// Kwa maneno mengine, nambari ya jozi ambayo ni 1 ikifuatiwa na sufuri 100.
	Big = 1 << 100
	// Ihamishe tena kwenda kulia nafasi 99, ili tupate 1<<1, au 2.
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
