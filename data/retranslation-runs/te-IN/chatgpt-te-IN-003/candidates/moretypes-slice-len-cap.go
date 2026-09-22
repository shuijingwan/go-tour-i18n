//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// స్లైస్‌కు సున్నా పొడవు వచ్చేలా దానిని స్లైస్ చేయండి.
	s = s[:0]
	printSlice(s)

	// దాని పొడవును పెంచండి.
	s = s[:4]
	printSlice(s)

	// దాని మొదటి రెండు విలువలను తొలగించండి.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
