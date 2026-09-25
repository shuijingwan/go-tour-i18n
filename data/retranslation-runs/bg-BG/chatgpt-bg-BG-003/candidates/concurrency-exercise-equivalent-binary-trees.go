//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk обхожда дървото t, като изпраща всички стойности
// от дървото към канала ch.
func Walk(t *tree.Tree, ch chan int)

// Same определя дали дърветата
// t1 и t2 съдържат едни и същи стойности.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
