//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i-ге көрсеткіш орнату
	fmt.Println(*p) // көрсеткіш арқылы i-ді оқу
	*p = 21         // көрсеткіш арқылы i-ді орнату
	fmt.Println(i)  // i-дің жаңа мәнін көру

	p = &j         // j-ге көрсеткіш орнату
	*p = *p / 37   // көрсеткіш арқылы j-ді бөлу
	fmt.Println(j) // j-дің жаңа мәнін көру
}
