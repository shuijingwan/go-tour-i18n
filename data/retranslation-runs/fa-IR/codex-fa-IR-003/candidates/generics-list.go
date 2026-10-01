//go:build OMIT

package main

// List یک فهرست پیوندی یک‌سویه را نمایش می‌دهد که
// مقادیری از هر نوع را نگه می‌دارد.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
