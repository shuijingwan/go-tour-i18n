//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // indică spre i
	fmt.Println(*p) // citește i prin pointer
	*p = 21         // setează i prin pointer
	fmt.Println(i)  // vezi noua valoare a lui i

	p = &j         // indică spre j
	*p = *p / 37   // împarte j prin pointer
	fmt.Println(j) // vezi noua valoare a lui j
}
