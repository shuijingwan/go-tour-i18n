//go:build OMIT

package main

// List एक सिंगली-लिंक्ड लिस्ट को दर्शाता है, जो
// किसी भी टाइप की वैल्यू रखती है।
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
