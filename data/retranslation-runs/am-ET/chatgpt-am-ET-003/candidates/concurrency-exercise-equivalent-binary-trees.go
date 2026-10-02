//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk በዛፉ t ውስጥ እየተጓዘ ሁሉንም እሴቶች ይልካል
// ከዛፉ ወደ ቻናል ch።
func Walk(t *tree.Tree, ch chan int)

// Same ዛፎቹ
// t1 እና t2 ተመሳሳይ እሴቶችን እንደያዙ ይወስናል።
func Same(t1, t2 *tree.Tree) bool

func main() {
}
