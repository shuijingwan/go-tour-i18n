//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, t എന്ന ട്രീയിലൂടെ സഞ്ചരിച്ച് എല്ലാ മൂല്യങ്ങളും
// tree-യിൽ നിന്ന് ch എന്ന ചാനലിലേക്ക് അയയ്ക്കുന്നു.
func Walk(t *tree.Tree, ch chan int)

// Same, രണ്ട് ട്രീകളിലും
// t1, t2 എന്നിവയിൽ ഒരേ മൂല്യങ്ങളാണോ ഉള്ളതെന്ന് നിർണയിക്കുന്നു.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
