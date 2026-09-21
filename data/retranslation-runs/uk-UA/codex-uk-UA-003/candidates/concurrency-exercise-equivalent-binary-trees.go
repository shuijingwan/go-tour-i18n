//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk обходить дерево t, надсилаючи всі значення
// з дерева до каналу ch.
func Walk(t *tree.Tree, ch chan int)

// Same визначає, чи містять дерева
// t1 і t2 однакові значення.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
