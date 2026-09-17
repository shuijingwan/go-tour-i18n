//go:build OMIT

package main

import "fmt"

const (
	// Buat bilangan sangat besar dengan menggeser bit 1 ke kiri sebanyak 100 posisi.
	// Dengan kata lain, bilangan biner berupa 1 yang diikuti 100 nol.
	Big = 1 << 100
	// Geser kembali ke kanan sebanyak 99 posisi, sehingga hasilnya menjadi 1<<1, atau 2.
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
