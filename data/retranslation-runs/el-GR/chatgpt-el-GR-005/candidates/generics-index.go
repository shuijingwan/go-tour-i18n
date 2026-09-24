//go:build OMIT

package main

import "fmt"

// Η Index επιστρέφει τον δείκτη θέσης του x στο s ή -1 αν δεν βρεθεί.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// Οι v και x είναι τύπου T, ο οποίος ικανοποιεί τον περιορισμό comparable.
		// Επομένως μπορούμε να χρησιμοποιήσουμε εδώ τον τελεστή ==.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Η Index λειτουργεί με ένα τμήμα πίνακα ακεραίων.
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Η Index λειτουργεί επίσης με ένα τμήμα πίνακα συμβολοσειρών.
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
