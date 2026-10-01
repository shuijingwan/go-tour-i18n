//go:build OMIT

package main

// List кез келген түрдегі
// мәндерді сақтайтын бір бағытты байланысқан тізімді білдіреді.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
