//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // peka på i
	fmt.Println(*p) // läs i via pekaren
	*p = 21         // tilldela i via pekaren
	fmt.Println(i)  // visa det nya värdet för i

	p = &j         // peka på j
	*p = *p / 37   // dividera j via pekaren
	fmt.Println(j) // visa det nya värdet för j
}
