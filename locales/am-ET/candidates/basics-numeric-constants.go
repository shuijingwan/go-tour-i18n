//go:build OMIT

package main

import "fmt"

const (
	// 1 ቢትን 100 ቦታዎች ወደ ግራ በማንሸራተት እጅግ ትልቅ ቁጥር ፍጠር።
	// በሌላ አገላለጽ፣ 1 ከዚያም 100 ዜሮዎች የሚከተሉት የሁለትዮሽ ቁጥር ነው።
	Big = 1 << 100
	// እንደገና 99 ቦታዎች ወደ ቀኝ አንሸራትተው፣ በመጨረሻ 1<<1 ወይም 2 እናገኛለን።
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
