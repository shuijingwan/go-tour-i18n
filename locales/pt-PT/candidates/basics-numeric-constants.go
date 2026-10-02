//go:build OMIT

package main

import "fmt"

const (
	// Crie um número enorme deslocando um bit 1 100 posições para a esquerda.
	// Por outras palavras, o número binário constituído por 1 seguido de 100 zeros.
	Big = 1 << 100
	// Desloque-o novamente 99 posições para a direita, para obtermos 1<<1, ou seja, 2.
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
