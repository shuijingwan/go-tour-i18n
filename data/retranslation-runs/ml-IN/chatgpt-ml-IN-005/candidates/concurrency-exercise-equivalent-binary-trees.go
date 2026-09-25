//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, t എന്ന ട്രീയിലൂടെ സഞ്ചരിച്ച് അതിലെ എല്ലാ മൂല്യങ്ങളും
// ch എന്ന ചാനലിലേക്ക് അയയ്ക്കുന്നു.
func Walk(t *tree.Tree, ch chan int)

// Same, t1, t2 എന്നീ ട്രീകളിൽ
// ഒരേ മൂല്യങ്ങളാണോ ഉള്ളതെന്ന് നിർണയിക്കുന്നു.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
