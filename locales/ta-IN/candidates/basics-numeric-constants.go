//go:build OMIT

package main

import "fmt"

const (
	// 1 பிட்-ஐ 100 இடங்கள் இடப்புறம் நகர்த்தி மிகப் பெரிய எண்ணை உருவாக்குங்கள்.
	// வேறு வார்த்தைகளில், 1-க்கு பின்னர் 100 பூஜ்யங்கள் வரும் இரும எண்.
	Big = 1 << 100
	// மீண்டும் அதை 99 இடங்கள் வலப்புறம் நகர்த்துங்கள்; அப்போது 1<<1, அதாவது 2 கிடைக்கும்.
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
