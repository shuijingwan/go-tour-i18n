//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk obilazi stablo t i šalje sve vrijednosti
// iz stabla u kanal ch.
func Walk(t *tree.Tree, ch chan int)

// Same utvrđuje sadržavaju li stabla
// t1 i t2 iste vrijednosti.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
