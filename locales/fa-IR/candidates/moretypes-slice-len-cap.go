//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// اسلایس را برش دهید تا طول آن صفر شود.
	s = s[:0]
	printSlice(s)

	// طول آن را افزایش دهید.
	s = s[:4]
	printSlice(s)

	// دو مقدار نخست آن را حذف کنید.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
