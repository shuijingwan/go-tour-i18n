//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // mutass i-re
	fmt.Println(*p) // olvasd i értékét a mutatón keresztül
	*p = 21         // állítsd be i értékét a mutatón keresztül
	fmt.Println(i)  // nézd meg i új értékét

	p = &j         // mutass j-re
	*p = *p / 37   // oszd el j értékét a mutatón keresztül
	fmt.Println(j) // nézd meg j új értékét
}
