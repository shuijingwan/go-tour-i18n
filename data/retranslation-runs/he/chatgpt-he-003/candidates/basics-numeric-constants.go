//go:build OMIT

package main

import "fmt"

const (
	// צרו מספר עצום באמצעות הזזה של הסיבית 1 שמאלה ב־100 מקומות.
	// במילים אחרות, זהו המספר הבינארי שמתחיל ב־1 ואחריו 100 אפסים.
	Big = 1 << 100
	// הזיזו אותו שוב ימינה ב־99 מקומות, כך שנקבל 1<<1, כלומר 2.
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
