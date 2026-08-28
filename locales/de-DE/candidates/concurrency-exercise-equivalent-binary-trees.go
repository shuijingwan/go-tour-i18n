//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk durchläuft den Baum t und sendet alle Werte
// aus dem Baum an den Kanal ch.
func Walk(t *tree.Tree, ch chan int)

// Same ermittelt, ob die Bäume
// t1 und t2 dieselben Werte enthalten.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
