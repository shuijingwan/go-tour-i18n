//go:build OMIT

package main

import "fmt"

const (
	// 1 हा bit 100 स्थानांनी डावीकडे shift करून मोठी संख्या तयार करा.
	// म्हणजे, 1 नंतर 100 शून्ये असलेली binary संख्या.
	Big = 1 << 100
	// ती पुन्हा 99 स्थानांनी उजवीकडे shift करा, म्हणजे शेवटी 1<<1, अर्थात 2, मिळेल.
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
