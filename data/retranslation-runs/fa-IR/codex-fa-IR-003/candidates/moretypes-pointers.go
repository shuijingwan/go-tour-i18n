//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // اشاره به i
	fmt.Println(*p) // خواندن i از طریق اشاره‌گر
	*p = 21         // تنظیم i از طریق اشاره‌گر
	fmt.Println(i)  // دیدن مقدار جدید i

	p = &j         // اشاره به j
	*p = *p / 37   // تقسیم j از طریق اشاره‌گر
	fmt.Println(j) // دیدن مقدار جدید j
}
