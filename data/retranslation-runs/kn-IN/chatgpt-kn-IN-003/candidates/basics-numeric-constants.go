//go:build OMIT

package main

import "fmt"

const (
	// 1 ಬಿಟ್ ಅನ್ನು 100 ಸ್ಥಾನಗಳಷ್ಟು ಎಡಕ್ಕೆ ಶಿಫ್ಟ್ ಮಾಡಿ ಅತಿ ದೊಡ್ಡ ಸಂಖ್ಯೆಯನ್ನು ರಚಿಸಿ.
	// ಅಂದರೆ, 1 ರ ನಂತರ 100 ಸೊನ್ನೆಗಳಿರುವ ದ್ವಿಮಾನ ಸಂಖ್ಯೆ.
	Big = 1 << 100
	// ನಂತರ ಅದನ್ನು 99 ಸ್ಥಾನಗಳಷ್ಟು ಬಲಕ್ಕೆ ಶಿಫ್ಟ್ ಮಾಡಿ; ಆಗ 1<<1, ಅಂದರೆ 2 ಸಿಗುತ್ತದೆ.
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
