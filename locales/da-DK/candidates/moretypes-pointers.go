//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // peg på i
	fmt.Println(*p) // læs i via pointeren
	*p = 21         // ændr i via pointeren
	fmt.Println(i)  // se den nye værdi af i

	p = &j         // peg på j
	*p = *p / 37   // divider j via pointeren
	fmt.Println(j) // se den nye værdi af j
}
