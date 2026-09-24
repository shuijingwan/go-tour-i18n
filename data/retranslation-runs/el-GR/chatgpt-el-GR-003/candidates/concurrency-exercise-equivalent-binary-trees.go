//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Η Walk διατρέχει το δέντρο t και στέλνει όλες τις τιμές
// από το δέντρο στο κανάλι ch.
func Walk(t *tree.Tree, ch chan int)

// Η Same καθορίζει αν τα δέντρα
// t1 και t2 περιέχουν τις ίδιες τιμές.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
