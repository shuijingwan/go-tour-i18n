//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // ituro ang pointer sa i
	fmt.Println(*p) // basahin ang i sa pamamagitan ng pointer
	*p = 21         // itakda ang i sa pamamagitan ng pointer
	fmt.Println(i)  // tingnan ang bagong value ng i

	p = &j         // ituro ang pointer sa j
	*p = *p / 37   // hatiin ang j sa pamamagitan ng pointer
	fmt.Println(j) // tingnan ang bagong value ng j
}
