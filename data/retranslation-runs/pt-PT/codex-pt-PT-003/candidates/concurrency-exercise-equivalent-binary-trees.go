//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk percorre a árvore t, enviando todos os valores
// da árvore para o canal ch.
func Walk(t *tree.Tree, ch chan int)

// Same determina se as árvores
// t1 e t2 contêm os mesmos valores.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
