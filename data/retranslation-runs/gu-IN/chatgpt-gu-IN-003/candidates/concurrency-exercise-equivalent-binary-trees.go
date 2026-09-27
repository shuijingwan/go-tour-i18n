//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, ટ્રી t માંથી પસાર થઈને તેના બધાં મૂલ્યો
// ટ્રીમાંથી ચેનલ ch પર મોકલે છે.
func Walk(t *tree.Tree, ch chan int)

// Same તપાસે છે કે બંને ટ્રીમાં
// t1 અને t2 માં એકસરખાં મૂલ્યો છે કે નહીં.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
