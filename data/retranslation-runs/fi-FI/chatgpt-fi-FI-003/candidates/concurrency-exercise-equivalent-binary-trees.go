//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk käy läpi puun t ja lähettää kaikki arvot
// puusta kanavaan ch.
func Walk(t *tree.Tree, ch chan int)

// Same selvittää, sisältävätkö puut
// t1 ja t2 samat arvot.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
