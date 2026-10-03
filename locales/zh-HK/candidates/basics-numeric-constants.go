//go:build OMIT

package main

import "fmt"

const (
	// 將值為 1 的位元向左移動 100 位，以建立一個巨大數字。
	// 換言之，就是 1 後面接着 100 個零的二進制數字。
	Big = 1 << 100
	// 再將它向右移動 99 位，最終得到 1<<1，也就是 2。
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
