//go:build OMIT

package main

import "fmt"

const (
	// 1 비트를 왼쪽으로 100자리만큼 시프트하여 아주 큰 수를 만듭니다.
	// 다시 말해, 1 뒤에 0이 100개 이어지는 이진수입니다.
	Big = 1 << 100
	// 다시 오른쪽으로 99자리만큼 시프트하면 1<<1, 즉 2가 됩니다.
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
