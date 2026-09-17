//go:build OMIT

package main

import "fmt"

const (
	// 建立一個巨大的數字，將 1 位元向左位移 100 位。
	// 換句話說，就是 1 後面接著 100 個 0 的二進位數。
	Big = 1 << 100
	// 再向右位移 99 位，因此最後得到 1<<1，也就是 2。
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
