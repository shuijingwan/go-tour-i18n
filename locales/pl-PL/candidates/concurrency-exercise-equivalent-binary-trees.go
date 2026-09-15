//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk przechodzi przez drzewo t, wysyłając wszystkie wartości
// z drzewa do kanału ch.
func Walk(t *tree.Tree, ch chan int)

// Same określa, czy drzewa
// t1 i t2 zawierają te same wartości.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
