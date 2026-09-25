//go:build OMIT

package main

// List എന്നത് ഒറ്റദിശയിൽ ബന്ധിപ്പിച്ച ലിസ്റ്റാണ്; അതിൽ
// ഏതു ടൈപ്പിലുള്ള മൂല്യങ്ങളും സൂക്ഷിക്കാം.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
