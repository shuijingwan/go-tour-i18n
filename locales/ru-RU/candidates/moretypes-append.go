//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append работает с nil-срезами.
	s = append(s, 0)
	printSlice(s)

	// Срез увеличивается по мере необходимости.
	s = append(s, 1)
	printSlice(s)

	// За один раз можно добавить несколько элементов.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
