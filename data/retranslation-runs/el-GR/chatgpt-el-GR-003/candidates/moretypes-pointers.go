//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // να δείχνει στην i
	fmt.Println(*p) // ανάγνωση της i μέσω του δείκτη
	*p = 21         // αλλαγή της i μέσω του δείκτη
	fmt.Println(i)  // δείτε τη νέα τιμή της i

	p = &j         // να δείχνει στην j
	*p = *p / 37   // διαίρεση της j μέσω του δείκτη
	fmt.Println(j) // δείτε τη νέα τιμή της j
}
