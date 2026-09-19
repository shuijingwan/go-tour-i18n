//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append nil स्लाइस पर काम करता है।
	s = append(s, 0)
	printSlice(s)

	// आवश्यकतानुसार स्लाइस बढ़ती है।
	s = append(s, 1)
	printSlice(s)

	// हम एक समय में एक से अधिक एलिमेंट जोड़ सकते हैं।
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
