//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // tunjuk ke i
	fmt.Println(*p) // baca i melalui penunjuk
	*p = 21         // tetapkan i melalui penunjuk
	fmt.Println(i)  // lihat nilai baharu i

	p = &j         // tunjuk ke j
	*p = *p / 37   // bahagikan j melalui penunjuk
	fmt.Println(j) // lihat nilai baharu j
}
