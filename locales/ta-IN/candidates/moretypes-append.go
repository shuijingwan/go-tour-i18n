//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append, nil துண்டங்களில் செயல்படுகிறது.
	s = append(s, 0)
	printSlice(s)

	// தேவைக்கேற்ப துண்டம் வளர்கிறது.
	s = append(s, 1)
	printSlice(s)

	// ஒரே நேரத்தில் ஒன்றுக்கு மேற்பட்ட உறுப்புகளைச் சேர்க்கலாம்.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
