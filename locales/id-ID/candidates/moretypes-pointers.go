//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // menunjuk ke i
	fmt.Println(*p) // baca i melalui pointer
	*p = 21         // tetapkan i melalui pointer
	fmt.Println(i)  // lihat nilai baru i

	p = &j         // menunjuk ke j
	*p = *p / 37   // bagi j melalui pointer
	fmt.Println(j) // lihat nilai baru j
}
