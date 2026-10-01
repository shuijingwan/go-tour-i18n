//go:build OMIT

package main

import "fmt"

// Index اندیس x در s را برمی‌گرداند؛ اگر پیدا نشود، -1 را برمی‌گرداند.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v و x از نوع T هستند که قید comparable را دارد،
		// بنابراین اینجا می‌توانیم از == استفاده کنیم.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index روی اسلایسی از intها کار می‌کند
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index روی اسلایسی از stringها نیز کار می‌کند
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
