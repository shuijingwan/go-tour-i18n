//go:build OMIT

package main

// List የሚወክለው ነጠላ-ተያያዥ ዝርዝር ሲሆን የሚይዘው
// የማንኛውንም ዓይነት እሴቶች ነው።
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
