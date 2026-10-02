//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // указать на i
	fmt.Println(*p) // прочитать i через указатель
	*p = 21         // изменить i через указатель
	fmt.Println(i)  // посмотреть новое значение i

	p = &j         // указать на j
	*p = *p / 37   // разделить j через указатель
	fmt.Println(j) // посмотреть новое значение j
}
