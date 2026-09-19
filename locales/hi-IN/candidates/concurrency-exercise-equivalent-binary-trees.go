//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, ट्री t पर चलते हुए सभी वैल्यू भेजता है
// ट्री से चैनल ch में।
func Walk(t *tree.Tree, ch chan int)

// Same यह निर्धारित करता है कि ट्री
// t1 और t2 में समान वैल्यू हैं या नहीं।
func Same(t1, t2 *tree.Tree) bool

func main() {
}
