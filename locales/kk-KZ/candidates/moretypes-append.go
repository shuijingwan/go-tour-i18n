//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append nil слайстармен жұмыс істейді.
	s = append(s, 0)
	printSlice(s)

	// Слайс қажетіне қарай ұлғаяды.
	s = append(s, 1)
	printSlice(s)

	// Бір уақытта бірден көп элемент қоса аламыз.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
