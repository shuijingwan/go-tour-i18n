//go:build OMIT

package main

import "fmt"

const (
	// 1 బిట్‌ను 100 స్థానాలు ఎడమవైపు షిఫ్ట్ చేసి ఒక చాలా పెద్ద సంఖ్యను సృష్టించండి.
	// మరో మాటలో చెప్పాలంటే, 1 తర్వాత 100 సున్నాలు ఉండే బైనరీ సంఖ్య.
	Big = 1 << 100
	// దాన్ని మళ్లీ 99 స్థానాలు కుడివైపు షిఫ్ట్ చేస్తే, చివరికి 1<<1, అంటే 2 వస్తుంది.
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
