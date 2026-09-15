//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk traverserar trädet t och skickar alla värden
// från trädet till kanalen ch.
func Walk(t *tree.Tree, ch chan int)

// Same avgör om träden
// t1 och t2 innehåller samma värden.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
