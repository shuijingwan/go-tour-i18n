//go:build OMIT

package main

import "fmt"

const (
	// 将一个 1 位向左移动 100 位，创建一个巨大的数字。
	// 换句话说，就是一个 1 后面跟着 100 个 0 的二进制数。
	Big = 1 << 100
	// 再将它向右移动 99 位，最终得到 1<<1，也就是 2。
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
