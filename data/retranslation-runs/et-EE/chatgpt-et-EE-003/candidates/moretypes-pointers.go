//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // viit osutab muutujale i
	fmt.Println(*p) // loe i väärtus viida kaudu
	*p = 21         // määra i väärtus viida kaudu
	fmt.Println(i)  // vaata i uut väärtust

	p = &j         // viit osutab muutujale j
	*p = *p / 37   // jaga j väärtus viida kaudu
	fmt.Println(j) // vaata j uut väärtust
}
