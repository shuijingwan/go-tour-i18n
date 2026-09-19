//go:build OMIT

package main

// List এমন একটি সিঙ্গলি-লিঙ্কড লিস্টকে উপস্থাপন করে, যা
// যেকোনো টাইপের মান ধারণ করে।
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
