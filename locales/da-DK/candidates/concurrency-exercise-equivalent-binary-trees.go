//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk gennemløber træet t og sender alle værdierne
// fra træet til kanalen ch.
func Walk(t *tree.Tree, ch chan int)

// Same afgør, om træerne
// t1 og t2 indeholder de samme værdier.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
