//go:build OMIT

package main

import "fmt"

// Index, s లో x యొక్క ఇండెక్స్‌ను రిటర్న్ చేస్తుంది; కనబడకపోతే -1 ను రిటర్న్ చేస్తుంది.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v మరియు x టైప్ T కు చెందినవి; T కు comparable
		// కన్‌స్ట్రెయింట్ ఉంది, కాబట్టి ఇక్కడ == ను ఉపయోగించవచ్చు.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index ints స్లైస్‌పై పనిచేస్తుంది
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index strings స్లైస్‌పై కూడా పనిచేస్తుంది
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
