//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk, ట్రీ t లోని అన్ని విలువలను
// చానల్ ch కు పంపుతూ ఆ ట్రీలో సంచరిస్తుంది.
func Walk(t *tree.Tree, ch chan int)

// Same అనేది t1 మరియు t2 అనే రెండు ట్రీల్లో
// ఒకే విలువలు ఉన్నాయో లేదో నిర్ధారిస్తుంది.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
