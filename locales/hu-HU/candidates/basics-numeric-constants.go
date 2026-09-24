//go:build OMIT

package main

import "fmt"

const (
	// Hozz létre egy hatalmas számot úgy, hogy az 1 bitet 100 hellyel balra tolod.
	// Más szóval: ez a kettes számrendszerbeli szám egy 1-esből és utána 100 nullából áll.
	Big = 1 << 100
	// Told vissza 99 hellyel jobbra, így végül 1<<1, azaz 2 marad.
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
