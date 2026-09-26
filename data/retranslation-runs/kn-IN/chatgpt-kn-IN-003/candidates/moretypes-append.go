//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append ಫಂಕ್ಷನ್ nil ಸ್ಲೈಸ್‌ಗಳ ಮೇಲೂ ಕೆಲಸ ಮಾಡುತ್ತದೆ.
	s = append(s, 0)
	printSlice(s)

	// ಅಗತ್ಯಕ್ಕೆ ಅನುಗುಣವಾಗಿ ಸ್ಲೈಸ್‌ನ ಉದ್ದ ಹೆಚ್ಚುತ್ತದೆ.
	s = append(s, 1)
	printSlice(s)

	// ಒಂದೇ ಬಾರಿ ಒಂದಕ್ಕಿಂತ ಹೆಚ್ಚು ಅಂಶಗಳನ್ನು ಸೇರಿಸಬಹುದು.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
