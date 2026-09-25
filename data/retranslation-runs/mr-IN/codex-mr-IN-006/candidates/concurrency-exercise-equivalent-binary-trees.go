//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk फंक्शन t वृक्षावरून चालत त्यातील सर्व मूल्ये
// वृक्षातून चॅनेल ch कडे पाठवते.
func Walk(t *tree.Tree, ch chan int)

// Same फंक्शन t1 आणि t2 या वृक्षांमध्ये
// समान मूल्ये आहेत का हे ठरवते.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
