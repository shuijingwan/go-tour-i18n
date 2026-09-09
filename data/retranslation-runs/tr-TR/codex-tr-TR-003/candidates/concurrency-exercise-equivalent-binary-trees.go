//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, t ağacını dolaşıp tüm değerleri
// ch kanalına gönderir.
func Walk(t *tree.Tree, ch chan int)

// Same,
// t1 ve t2 ağaçlarının aynı değerleri içerip içermediğini belirler.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
