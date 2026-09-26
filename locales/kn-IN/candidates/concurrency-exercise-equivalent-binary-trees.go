//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk ಫಂಕ್ಷನ್ t ಟ್ರೀಯಲ್ಲಿ ಸಂಚರಿಸಿ ಅದರ ಎಲ್ಲ ಮೌಲ್ಯಗಳನ್ನು
// ಟ್ರೀಯಿಂದ ch ಚಾನೆಲ್‌ಗೆ ಕಳುಹಿಸುತ್ತದೆ.
func Walk(t *tree.Tree, ch chan int)

// Same ಫಂಕ್ಷನ್ t1 ಮತ್ತು t2 ಎಂಬ ಟ್ರೀಗಳು
// ಒಂದೇ ಮೌಲ್ಯಗಳನ್ನು ಹೊಂದಿವೆಯೇ ಎಂದು ನಿರ್ಧರಿಸುತ್ತದೆ.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
