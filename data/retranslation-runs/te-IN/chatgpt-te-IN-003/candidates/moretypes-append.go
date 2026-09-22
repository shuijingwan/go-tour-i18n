//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append, nil స్లైస్‌లపై పనిచేస్తుంది.
	s = append(s, 0)
	printSlice(s)

	// అవసరమైన మేరకు స్లైస్ పెరుగుతుంది.
	s = append(s, 1)
	printSlice(s)

	// ఒకేసారి ఒకటి కంటే ఎక్కువ ఎలిమెంట్‌లను జోడించవచ్చు.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
