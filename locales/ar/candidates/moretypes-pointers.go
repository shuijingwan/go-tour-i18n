//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // أشر إلى i
	fmt.Println(*p) // اقرأ i عبر المؤشر
	*p = 21         // عيّن i عبر المؤشر
	fmt.Println(i)  // شاهد القيمة الجديدة لـ i

	p = &j         // أشر إلى j
	*p = *p / 37   // اقسم j عبر المؤشر
	fmt.Println(j) // شاهد القيمة الجديدة لـ j
}
