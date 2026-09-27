//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// ਸਲਾਈਸ ਨੂੰ ਕੱਟ ਕੇ ਉਸ ਦੀ ਲੰਬਾਈ ਸਿਫ਼ਰ ਕਰੋ।
	s = s[:0]
	printSlice(s)

	// ਇਸ ਦੀ ਲੰਬਾਈ ਵਧਾਓ।
	s = s[:4]
	printSlice(s)

	// ਇਸ ਦੇ ਪਹਿਲੇ ਦੋ ਮੁੱਲ ਹਟਾਓ।
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
