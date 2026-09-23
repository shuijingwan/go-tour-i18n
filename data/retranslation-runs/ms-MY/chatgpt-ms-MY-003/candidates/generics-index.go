//go:build OMIT

package main

import "fmt"

// Index mengembalikan indeks x dalam s, atau -1 jika tidak ditemui.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v dan x berjenis T, yang mempunyai
		// kekangan comparable, jadi kita boleh menggunakan == di sini.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index berfungsi pada hirisan int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index juga berfungsi pada hirisan rentetan
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
