//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk menjelajah pokok t sambil menghantar semua nilai
// daripada pokok ke saluran ch.
func Walk(t *tree.Tree, ch chan int)

// Same menentukan sama ada pokok
// t1 dan t2 mengandungi nilai yang sama.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
