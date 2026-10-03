//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk parcourt l’arbre t en envoyant toutes les valeurs
// de l’arbre vers le canal ch.
func Walk(t *tree.Tree, ch chan int)

// Same détermine si les arbres
// t1 et t2 contiennent les mêmes valeurs.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
