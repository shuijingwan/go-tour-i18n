//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk tree t मध्ये फिरून सर्व मूल्ये
// tree मधून channel ch कडे पाठवतो.
func Walk(t *tree.Tree, ch chan int)

// Same हे trees
// t1 आणि t2 मध्ये तीच मूल्ये आहेत का हे ठरवते.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
