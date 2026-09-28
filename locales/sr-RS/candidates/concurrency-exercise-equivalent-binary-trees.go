//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk обилази стабло t, шаљући све вредности
// из стабла на канал ch.
func Walk(t *tree.Tree, ch chan int)

// Same утврђује да ли стабла
// t1 и t2 садрже исте вредности.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
