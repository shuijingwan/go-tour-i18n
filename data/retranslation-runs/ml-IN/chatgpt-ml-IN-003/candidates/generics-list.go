//go:build OMIT

package main

// List, ഏകദിശ ലിങ്ക്ഡ് ലിസ്റ്റിനെ പ്രതിനിധീകരിക്കുന്നു; അതിൽ
// ഏതു ടൈപ്പിലുള്ള മൂല്യങ്ങളും സൂക്ഷിക്കാം.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
