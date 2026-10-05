//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // הצביעו אל i
	fmt.Println(*p) // קראו את i דרך המצביע
	*p = 21         // שנו את הערך של i דרך המצביע
	fmt.Println(i)  // ראו את הערך החדש של i

	p = &j         // הצביעו אל j
	*p = *p / 37   // חלקו את j דרך המצביע
	fmt.Println(j) // ראו את הערך החדש של j
}
