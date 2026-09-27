//go:build OMIT

package main

import "fmt"

// Index, s ਵਿੱਚ x ਦਾ ਇੰਡੈਕਸ ਵਾਪਸ ਕਰਦਾ ਹੈ ਜਾਂ ਨਾ ਮਿਲਣ ਉੱਤੇ -1 ਵਾਪਸ ਕਰਦਾ ਹੈ।
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ਅਤੇ x ਦੀ ਟਾਈਪ T ਹੈ, ਜਿਸ ਉੱਤੇ comparable
		// ਕੰਸਟ੍ਰੇਇੰਟ ਹੈ, ਇਸ ਲਈ ਅਸੀਂ ਇੱਥੇ == ਵਰਤ ਸਕਦੇ ਹਾਂ।
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index, ints ਦੀ ਸਲਾਈਸ ਉੱਤੇ ਕੰਮ ਕਰਦਾ ਹੈ
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index, strings ਦੀ ਸਲਾਈਸ ਉੱਤੇ ਵੀ ਕੰਮ ਕਰਦਾ ਹੈ
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
