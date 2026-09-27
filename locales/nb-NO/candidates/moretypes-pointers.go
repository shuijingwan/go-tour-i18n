//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // pek på i
	fmt.Println(*p) // les i gjennom pekeren
	*p = 21         // endre i gjennom pekeren
	fmt.Println(i)  // vis den nye verdien av i

	p = &j         // pek på j
	*p = *p / 37   // divider j gjennom pekeren
	fmt.Println(j) // vis den nye verdien av j
}
