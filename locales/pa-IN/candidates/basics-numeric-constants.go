//go:build OMIT

package main

import "fmt"

const (
	// 1 ਬਿਟ ਨੂੰ 100 ਥਾਵਾਂ ਖੱਬੇ ਸ਼ਿਫਟ ਕਰਕੇ ਇੱਕ ਬਹੁਤ ਵੱਡੀ ਸੰਖਿਆ ਬਣਾਓ।
	// ਦੂਜੇ ਸ਼ਬਦਾਂ ਵਿੱਚ, ਉਹ ਬਾਈਨਰੀ ਸੰਖਿਆ ਜਿਸ ਵਿੱਚ 1 ਤੋਂ ਬਾਅਦ 100 ਸਿਫ਼ਰ ਹਨ।
	Big = 1 << 100
	// ਇਸ ਨੂੰ ਮੁੜ 99 ਥਾਵਾਂ ਸੱਜੇ ਸ਼ਿਫਟ ਕਰੋ, ਤਾਂ ਜੋ ਅੰਤ ਵਿੱਚ ਸਾਡੇ ਕੋਲ 1<<1, ਅਰਥਾਤ 2, ਰਹਿ ਜਾਵੇ।
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
