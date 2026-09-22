//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, t மரத்தில் நடந்து அதன் அனைத்து மதிப்புகளையும்
// அந்த மரத்திலிருந்து ch தடத்துக்கு அனுப்புகிறது.
func Walk(t *tree.Tree, ch chan int)

// Same, மரங்கள்
// t1 மற்றும் t2 ஒரே மதிப்புகளைக் கொண்டுள்ளனவா என்பதைத் தீர்மானிக்கிறது.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
