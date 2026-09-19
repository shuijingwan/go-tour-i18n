//go:build OMIT

package main

import "fmt"

const (
	// একটি 1 বিটকে 100 ঘর বামে সরিয়ে একটি বিশাল সংখ্যা তৈরি করুন।
	// অন্যভাবে বললে, এটি এমন একটি বাইনারি সংখ্যা যার শুরুতে 1 এবং পরে 100টি শূন্য থাকে।
	Big = 1 << 100
	// এটিকে আবার 99 ঘর ডানে সরান, তাহলে শেষে পাব 1<<1, অর্থাৎ 2।
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
