//go:build OMIT

package main

// List представлява едносвързан списък, който съхранява
// стойности от произволен тип.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
