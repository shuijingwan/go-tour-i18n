//go:build OMIT

package main

import "fmt"

const (
	// Skapa ett enormt tal genom att skifta biten 1 100 positioner åt vänster.
	// Med andra ord, det binära tal som består av 1 följt av 100 nollor.
	Big = 1 << 100
	// Skifta det 99 positioner åt höger igen, så får vi 1<<1, det vill säga 2.
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
