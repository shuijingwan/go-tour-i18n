//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk apstaigā koku t, nosūtot visas vērtības
// no koka uz kanālu ch.
func Walk(t *tree.Tree, ch chan int)

// Same nosaka, vai koki
// t1 un t2 satur vienādas vērtības.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
