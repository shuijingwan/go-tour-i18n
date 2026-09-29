//go:build OMIT

package main

import "fmt"

const (
	// Ustvarite ogromno število tako, da bit 1 premaknete za 100 mest v levo.
	// Z drugimi besedami, dvojiško število, ki mu za 1 sledi 100 ničel.
	Big = 1 << 100
	// Znova ga premaknite za 99 mest v desno, tako da dobimo 1<<1 oziroma 2.
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
