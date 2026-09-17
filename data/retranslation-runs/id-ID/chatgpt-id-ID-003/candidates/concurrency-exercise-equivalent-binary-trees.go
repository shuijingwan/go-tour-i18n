//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk menelusuri pohon t dan mengirim semua nilai
// dari pohon ke kanal ch.
func Walk(t *tree.Tree, ch chan int)

// Same menentukan apakah pohon
// t1 dan t2 berisi nilai yang sama.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
