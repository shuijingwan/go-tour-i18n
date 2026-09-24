//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// Η append λειτουργεί ακόμη και όταν το τμήμα πίνακα είναι nil.
	s = append(s, 0)
	printSlice(s)

	// Το τμήμα πίνακα μεγαλώνει όσο χρειάζεται.
	s = append(s, 1)
	printSlice(s)

	// Μπορούμε να προσθέσουμε περισσότερα από ένα στοιχεία κάθε φορά.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
