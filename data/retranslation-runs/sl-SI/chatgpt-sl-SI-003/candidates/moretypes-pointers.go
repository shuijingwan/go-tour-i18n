//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // kaži na i
	fmt.Println(*p) // preberi i prek kazalca
	*p = 21         // nastavi i prek kazalca
	fmt.Println(i)  // oglej si novo vrednost i

	p = &j         // kaži na j
	*p = *p / 37   // deli j prek kazalca
	fmt.Println(j) // oglej si novo vrednost j
}
