//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// ഈ മെഥഡ് ഉള്ളതിനാൽ T ടൈപ്പ് I ഇന്റർഫേസ് നടപ്പാക്കുന്നു,
// എന്നാൽ അത് അങ്ങനെ ചെയ്യുന്നതായി വ്യക്തമായി പ്രഖ്യാപിക്കേണ്ടതില്ല.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
