//go:build OMIT

package main

// List mewakili senarai berangkai tunggal yang menyimpan
// nilai daripada sebarang jenis.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
