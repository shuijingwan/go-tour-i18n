//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append, nil dilimlerinde çalışır.
	s = append(s, 0)
	printSlice(s)

	// Dilim gerektiği kadar büyür.
	s = append(s, 1)
	printSlice(s)

	// Bir defada birden fazla öğe ekleyebiliriz.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
