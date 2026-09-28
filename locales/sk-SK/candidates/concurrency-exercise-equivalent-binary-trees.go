//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk prechádza strom t a odosiela všetky hodnoty
// zo stromu do kanála ch.
func Walk(t *tree.Tree, ch chan int)

// Same určuje, či stromy
// t1 a t2 obsahujú rovnaké hodnoty.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
