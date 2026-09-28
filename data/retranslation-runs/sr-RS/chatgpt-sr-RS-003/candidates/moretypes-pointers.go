//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // показује на i
	fmt.Println(*p) // прочитајте i преко показивача
	*p = 21         // поставите i преко показивача
	fmt.Println(i)  // погледајте нову вредност i

	p = &j         // показује на j
	*p = *p / 37   // поделите j преко показивача
	fmt.Println(j) // погледајте нову вредност j
}
