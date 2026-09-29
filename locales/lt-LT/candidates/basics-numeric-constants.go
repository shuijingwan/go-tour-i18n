//go:build OMIT

package main

import "fmt"

const (
	// Sukurkite didžiulį skaičių, pastumdami 1 bitą į kairę per 100 pozicijų.
	// Kitaip tariant, tai dvejetainis skaičius, kuriame po 1 eina 100 nulių.
	Big = 1 << 100
	// Vėl pastumkite jį į dešinę per 99 pozicijas, kad gautume 1<<1, arba 2.
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
