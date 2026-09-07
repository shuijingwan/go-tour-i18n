//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // wijs naar i
	fmt.Println(*p) // lees i via de pointer
	*p = 21         // stel i in via de pointer
	fmt.Println(i)  // bekijk de nieuwe waarde van i

	p = &j         // wijs naar j
	*p = *p / 37   // deel j via de pointer
	fmt.Println(j) // bekijk de nieuwe waarde van j
}
