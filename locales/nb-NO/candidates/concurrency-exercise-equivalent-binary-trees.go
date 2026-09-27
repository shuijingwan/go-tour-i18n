//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk går gjennom treet t og sender alle verdiene
// fra treet til kanalen ch.
func Walk(t *tree.Tree, ch chan int)

// Same avgjør om trærne
// t1 og t2 inneholder de samme verdiene.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
