//go:build OMIT

package main

import "fmt"

// Index, s içinde bulunan x öğesinin indeksini, bulunamazsa -1 değerini döndürür.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ve x, comparable kısıtına sahip T türündedir;
		// bu nedenle burada == kullanabiliriz.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index, int değerlerinden oluşan bir dilimde çalışır
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index, string değerlerinden oluşan bir dilimde de çalışır
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
