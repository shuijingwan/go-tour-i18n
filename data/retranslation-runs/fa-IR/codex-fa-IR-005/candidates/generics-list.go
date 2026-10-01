//go:build OMIT

package main

// List نمایانگر یک فهرست پیوندی یک‌سویه است که
// مقادیری از هر نوع را نگه می‌دارد.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
