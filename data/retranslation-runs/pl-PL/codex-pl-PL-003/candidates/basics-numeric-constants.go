//go:build OMIT

package main

import "fmt"

const (
	// Utwórz ogromną liczbę, przesuwając bit 1 o 100 pozycji w lewo.
	// Innymi słowy, liczbę binarną złożoną z 1, po której następuje 100 zer.
	Big = 1 << 100
	// Przesuń ją ponownie o 99 pozycji w prawo, aby otrzymać 1<<1, czyli 2.
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
