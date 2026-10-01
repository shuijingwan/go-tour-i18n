//go:build OMIT

package main

// List inawakilisha orodha iliyounganishwa kwa kiungo kimoja inayohifadhi
// thamani za aina yoyote.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
