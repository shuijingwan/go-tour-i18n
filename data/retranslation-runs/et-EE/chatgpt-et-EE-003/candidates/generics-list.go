//go:build OMIT

package main

// List tähistab ühesuunalist ahelloendit, mis sisaldab
// mis tahes tüüpi väärtusi.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
