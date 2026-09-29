//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk pereina medį t ir siunčia visas reikšmes
// iš medžio į kanalą ch.
func Walk(t *tree.Tree, ch chan int)

// Same nustato, ar medžiuose
// t1 ir t2 yra tos pačios reikšmės.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
