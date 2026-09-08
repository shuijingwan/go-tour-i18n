//go:build OMIT

package main

import "fmt"

const (
	// Crie um número enorme deslocando um bit de valor 1 em 100 posições para a esquerda.
	// Em outras palavras, o número binário que consiste em 1 seguido por 100 zeros.
	Big = 1 << 100
	// Desloque-o novamente 99 posições para a direita, de modo que o resultado seja 1<<1, ou 2.
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
