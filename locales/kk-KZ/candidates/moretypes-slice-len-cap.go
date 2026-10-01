//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// Слайстың ұзындығын нөл ету үшін оны қайта кесіңіз.
	s = s[:0]
	printSlice(s)

	// Оның ұзындығын ұзартыңыз.
	s = s[:4]
	printSlice(s)

	// Алғашқы екі мәнін алып тастаңыз.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
