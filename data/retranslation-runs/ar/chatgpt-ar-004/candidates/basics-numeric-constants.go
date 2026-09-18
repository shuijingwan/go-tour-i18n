//go:build OMIT

package main

import "fmt"

const (
	// أنشئ عددًا هائلًا بإزاحة البت 1 إلى اليسار 100 موضع.
	// بعبارة أخرى، العدد الثنائي الذي يتكوّن من 1 يليه 100 صفر.
	Big = 1 << 100
	// أزحه إلى اليمين مرة أخرى 99 موضعًا، لننتهي إلى 1<<1، أي 2.
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
