//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk обходит дерево t, отправляя все значения
// из дерева в канал ch.
func Walk(t *tree.Tree, ch chan int)

// Same определяет, содержат ли деревья
// t1 и t2 одинаковые значения.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
