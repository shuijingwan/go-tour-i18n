//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// A Walk bejárja a t fát, és elküldi az összes értéket
// a fából a ch csatornára.
func Walk(t *tree.Tree, ch chan int)

// A Same megállapítja, hogy a
// t1 és t2 fák ugyanazokat az értékeket tartalmazzák-e.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
