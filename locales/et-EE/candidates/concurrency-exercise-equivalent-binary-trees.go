//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk läbib puud t ja saadab kõik väärtused
// puust kanalisse ch.
func Walk(t *tree.Tree, ch chan int)

// Same määrab, kas puud
// t1 ja t2 sisaldavad samu väärtusi.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
