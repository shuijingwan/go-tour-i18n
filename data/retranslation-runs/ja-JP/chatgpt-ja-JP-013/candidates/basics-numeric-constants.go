//go:build OMIT

package main

import "fmt"

const (
	// 1 を左に 100 ビットシフトして、巨大な数を作ります。
	// 言い換えると、1 の後に 100 個の 0 が続く二進数です。
	Big = 1 << 100
	// もう一度 99 ビット右にシフトすると、1<<1、つまり 2 になります。
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
