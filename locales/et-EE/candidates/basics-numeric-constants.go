//go:build OMIT

package main

import "fmt"

const (
	// Loo tohutu arv, nihutades bitti väärtusega 1 100 kohta vasakule.
	// Teisisõnu on see kahendarv, kus arvule 1 järgneb 100 nulli.
	Big = 1 << 100
	// Nihuta seda uuesti 99 kohta paremale, nii et tulemuseks jääb 1<<1 ehk 2.
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
