//go:build OMIT

package main

// A List egy egyszeresen láncolt listát képvisel, amely
// tetszőleges típusú értékeket tárol.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
