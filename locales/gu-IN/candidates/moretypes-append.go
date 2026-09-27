//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append, nil સ્લાઇસ પર કામ કરે છે.
	s = append(s, 0)
	printSlice(s)

	// જરૂર મુજબ સ્લાઇસનું કદ વધે છે.
	s = append(s, 1)
	printSlice(s)

	// એક સમયે એકથી વધુ ઘટક ઉમેરી શકાય છે.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
