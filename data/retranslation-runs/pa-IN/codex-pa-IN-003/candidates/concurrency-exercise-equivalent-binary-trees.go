//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk ਟ੍ਰੀ t ਉੱਤੇ ਚੱਲਦਾ ਹੋਇਆ ਸਾਰੇ ਮੁੱਲ
// ਟ੍ਰੀ ਤੋਂ ਚੈਨਲ ch ਵੱਲ ਭੇਜਦਾ ਹੈ।
func Walk(t *tree.Tree, ch chan int)

// Same ਨਿਰਧਾਰਤ ਕਰਦਾ ਹੈ ਕਿ ਟ੍ਰੀ
// t1 ਅਤੇ t2 ਵਿੱਚ ਇੱਕੋ ਮੁੱਲ ਹਨ ਜਾਂ ਨਹੀਂ।
func Same(t1, t2 *tree.Tree) bool

func main() {
}
