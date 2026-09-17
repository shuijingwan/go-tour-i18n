//go:build OMIT

package main

import "fmt"

// Index mengembalikan indeks x dalam s, atau -1 jika tidak ditemukan.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v dan x bertipe T, yang memiliki batasan tipe comparable
		// sehingga kita dapat menggunakan == di sini.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index bekerja pada slice int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index juga bekerja pada slice string
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
