//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// تجتاز Walk الشجرة t وترسل جميع القيم
// من الشجرة إلى القناة ch.
func Walk(t *tree.Tree, ch chan int)

// تحدد Same ما إذا كانت الشجرتان
// t1 وt2 تحتويان على القيم نفسها.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
