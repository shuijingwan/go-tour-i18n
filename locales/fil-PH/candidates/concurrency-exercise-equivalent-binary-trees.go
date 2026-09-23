//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Binabagtas ng Walk ang tree na t at ipinapadala ang lahat ng value
// mula sa tree patungo sa channel na ch.
func Walk(t *tree.Tree, ch chan int)

// Tinutukoy ng Same kung ang mga tree na
// t1 at t2 ay naglalaman ng parehong mga value.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
