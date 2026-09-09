//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i değişkenine işaret edin
	fmt.Println(*p) // işaretçi üzerinden i değerini okuyun
	*p = 21         // işaretçi üzerinden i değerini ayarlayın
	fmt.Println(i)  // i değişkeninin yeni değerine bakın

	p = &j         // j değişkenine işaret edin
	*p = *p / 37   // işaretçi üzerinden j değerini bölün
	fmt.Println(j) // j değişkeninin yeni değerine bakın
}
