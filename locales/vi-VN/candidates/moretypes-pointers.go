//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // trỏ tới i
	fmt.Println(*p) // đọc i thông qua con trỏ
	*p = 21         // gán i thông qua con trỏ
	fmt.Println(i)  // xem giá trị mới của i

	p = &j         // trỏ tới j
	*p = *p / 37   // chia j thông qua con trỏ
	fmt.Println(j) // xem giá trị mới của j
}
