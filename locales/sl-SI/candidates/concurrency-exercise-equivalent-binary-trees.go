//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk prehodi drevo t in pošilja vse vrednosti
// iz drevesa v kanal ch.
func Walk(t *tree.Tree, ch chan int)

// Same ugotovi, ali drevesi
// t1 in t2 vsebujeta enake vrednosti.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
