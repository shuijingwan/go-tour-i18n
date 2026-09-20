//go:build OMIT

package main

import "fmt"

const (
	// 1 bit کو 100 جگہ بائیں shift کر کے ایک بہت بڑا عدد بنائیں۔
	// دوسرے لفظوں میں، وہ binary عدد جس میں 1 کے بعد 100 صفر ہوں۔
	Big = 1 << 100
	// اسے دوبارہ 99 جگہ دائیں shift کریں، تاکہ آخر میں 1<<1، یعنی 2 رہ جائے۔
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
