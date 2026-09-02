//go:build OMIT

package main

import "fmt"

const (
	// 값 1을 왼쪽으로 100비트만큼 시프트하여 매우 큰 수를 만듭니다.
	// 다시 말해, 1 뒤에 0이 100개 이어지는 이진수입니다.
	Big = 1 << 100
	// 이 수를 다시 오른쪽으로 99비트만큼 시프트하면 결과는 1<<1, 즉 2입니다.
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
