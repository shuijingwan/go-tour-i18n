//go:build OMIT

package main

// List ਇੱਕ-ਦਿਸ਼ਾਈ ਕੜੀਬੱਧ ਸੂਚੀ ਨੂੰ ਦਰਸਾਉਂਦੀ ਹੈ ਜੋ
// ਕਿਸੇ ਵੀ ਟਾਈਪ ਦੇ ਮੁੱਲ ਰੱਖਦੀ ਹੈ।
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
