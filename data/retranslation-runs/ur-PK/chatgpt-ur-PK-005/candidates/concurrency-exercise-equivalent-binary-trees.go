//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk، tree t میں موجود تمام قدریں بھیجتا ہے
// tree سے چینل ch کو۔
func Walk(t *tree.Tree, ch chan int)

// Same یہ طے کرتا ہے کہ trees
// t1 اور t2 میں ایک جیسی قدریں ہیں یا نہیں۔
func Same(t1, t2 *tree.Tree) bool

func main() {
}
