//go:build OMIT

package main

// Η List αναπαριστά μια απλά συνδεδεμένη λίστα που περιέχει
// τιμές οποιουδήποτε τύπου.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
