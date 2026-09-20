//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// سلائس کو slice کر کے اس کی لمبائی صفر کریں۔
	s = s[:0]
	printSlice(s)

	// اس کی لمبائی بڑھائیں۔
	s = s[:4]
	printSlice(s)

	// اس کی پہلی دو قدریں ہٹا دیں۔
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
