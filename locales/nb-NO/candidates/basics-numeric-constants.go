//go:build OMIT

package main

import "fmt"

const (
	// Opprett et svært stort tall ved å forskyve biten 1 hundre plasser mot venstre.
	// Med andre ord: det binære tallet 1 etterfulgt av hundre nuller.
	Big = 1 << 100
	// Forskyv det deretter 99 plasser mot høyre, slik at vi ender opp med 1<<1, altså 2.
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
