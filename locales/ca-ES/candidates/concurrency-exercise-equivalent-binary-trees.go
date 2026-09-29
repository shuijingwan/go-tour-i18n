//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk recorre l’arbre t i envia tots els valors
// de l’arbre al canal ch.
func Walk(t *tree.Tree, ch chan int)

// Same determina si els arbres
// t1 i t2 contenen els mateixos valors.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
