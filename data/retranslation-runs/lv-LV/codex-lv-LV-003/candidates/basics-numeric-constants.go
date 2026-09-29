//go:build OMIT

package main

import "fmt"

const (
	// Izveido milzīgu skaitli, nobīdot 1 bitu pa kreisi par 100 pozīcijām.
	// Citiem vārdiem, tas ir binārs skaitlis, kuram aiz 1 seko 100 nulles.
	Big = 1 << 100
	// Nobīdi to atkal pa labi par 99 pozīcijām, iegūstot 1<<1 jeb 2.
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
