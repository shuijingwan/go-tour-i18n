//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk percorre l'albero t inviando tutti i valori
// dall'albero al canale ch.
func Walk(t *tree.Tree, ch chan int)

// Same determina se gli alberi
// t1 e t2 contengono gli stessi valori.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
