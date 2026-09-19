//go:build OMIT

package main

import "fmt"

const (
	// 1 बिट को 100 स्थान बाएँ शिफ्ट करके एक बहुत बड़ी संख्या बनाएँ।
	// दूसरे शब्दों में, वह बाइनरी संख्या जिसमें 1 के बाद 100 शून्य हों।
	Big = 1 << 100
	// इसे फिर 99 स्थान दाएँ शिफ्ट करें, ताकि अंत में 1<<1, यानी 2 मिले।
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
