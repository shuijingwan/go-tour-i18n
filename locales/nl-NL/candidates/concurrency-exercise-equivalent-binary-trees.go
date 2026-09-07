//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk doorloopt de boom t en stuurt alle waarden
// van de boom naar het kanaal ch.
func Walk(t *tree.Tree, ch chan int)

// Same bepaalt of de bomen
// t1 en t2 dezelfde waarden bevatten.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
