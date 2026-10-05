//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk עוברת על העץ t ושולחת את כל הערכים
// מהעץ לערוץ ch.
func Walk(t *tree.Tree, ch chan int)

// Same קובעת אם העצים
// t1 ו־t2 מכילים את אותם ערכים.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
