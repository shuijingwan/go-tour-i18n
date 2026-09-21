//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk parcurge arborele t și trimite toate valorile
// din arbore către canalul ch.
func Walk(t *tree.Tree, ch chan int)

// Same determină dacă arborii
// t1 și t2 conțin aceleași valori.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
