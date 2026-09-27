//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append, nil ਸਲਾਈਸਾਂ ਉੱਤੇ ਕੰਮ ਕਰਦਾ ਹੈ।
	s = append(s, 0)
	printSlice(s)

	// ਲੋੜ ਅਨੁਸਾਰ ਸਲਾਈਸ ਵਧਦੀ ਹੈ।
	s = append(s, 1)
	printSlice(s)

	// ਅਸੀਂ ਇੱਕ ਵਾਰ ਵਿੱਚ ਇੱਕ ਤੋਂ ਵੱਧ ਐਲੀਮੈਂਟ ਜੋੜ ਸਕਦੇ ਹਾਂ।
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
