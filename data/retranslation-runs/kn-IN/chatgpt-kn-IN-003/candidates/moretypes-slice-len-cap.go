//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// ಸ್ಲೈಸ್‌ನ ಉದ್ದ ಶೂನ್ಯವಾಗುವಂತೆ ಮತ್ತೊಮ್ಮೆ ಸ್ಲೈಸ್ ಮಾಡಿ.
	s = s[:0]
	printSlice(s)

	// ಅದರ ಉದ್ದವನ್ನು ಹೆಚ್ಚಿಸಿ.
	s = s[:4]
	printSlice(s)

	// ಅದರ ಮೊದಲ ಎರಡು ಮೌಲ್ಯಗಳನ್ನು ತೆಗೆದುಹಾಕಿ.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
