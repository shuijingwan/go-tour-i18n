//go:build OMIT

package main

import "fmt"

const (
	// 1 બિટને 100 સ્થાન ડાબે ખસેડીને ખૂબ મોટી સંખ્યા બનાવો.
	// બીજા શબ્દોમાં કહીએ તો, એવી બાઇનરી સંખ્યા જેમાં 1 પછી 100 શૂન્ય હોય.
	Big = 1 << 100
	// તેને ફરી 99 સ્થાન જમણે ખસેડો, જેથી અંતે 1<<1 એટલે કે 2 મળે.
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
