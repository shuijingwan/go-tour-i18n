//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk は木 t を走査し、そのすべての値を
// チャネル ch に送信します。
func Walk(t *tree.Tree, ch chan int)

// Same は、木
// t1 と t2 が同じ値を含むかどうかを判定します。
func Same(t1, t2 *tree.Tree) bool

func main() {
}
