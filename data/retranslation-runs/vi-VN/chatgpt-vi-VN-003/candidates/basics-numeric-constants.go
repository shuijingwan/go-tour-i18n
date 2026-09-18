//go:build OMIT

package main

import "fmt"

const (
	// Tạo một số rất lớn bằng cách dịch bit 1 sang trái 100 vị trí.
	// Nói cách khác, đó là số nhị phân gồm 1 theo sau bởi 100 số 0.
	Big = 1 << 100
	// Dịch nó sang phải 99 vị trí, để cuối cùng ta có 1<<1, tức là 2.
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
