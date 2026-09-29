//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // norāda uz i
	fmt.Println(*p) // nolasa i, izmantojot rādītāju
	*p = 21         // iestata i, izmantojot rādītāju
	fmt.Println(i)  // parāda jauno i vērtību

	p = &j         // norāda uz j
	*p = *p / 37   // dala j, izmantojot rādītāju
	fmt.Println(j) // parāda jauno j vērtību
}
