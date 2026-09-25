//go:build OMIT

package main

// List कोणत्याही टाइपची मूल्ये धारण करणाऱ्या एकदिश दुव्यांच्या यादीचे प्रतिनिधित्व करते
// .
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
