//go:build OMIT

package main

import "fmt"

const (
	// 1 हा बिट 100 स्थानांनी डावीकडे सरकवून मोठी संख्या तयार करा.
	// दुसऱ्या शब्दांत, 1 नंतर 100 शून्ये असलेली ही द्विमान संख्या आहे.
	Big = 1 << 100
	// आता ती पुन्हा 99 स्थानांनी उजवीकडे सरकवा; त्यामुळे शेवटी 1<<1, म्हणजे 2 मिळते.
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
