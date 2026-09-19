//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // ชี้ไปยัง i
	fmt.Println(*p) // อ่าน i ผ่านพอยน์เตอร์
	*p = 21         // กำหนดค่า i ผ่านพอยน์เตอร์
	fmt.Println(i)  // ดูค่าใหม่ของ i

	p = &j         // ชี้ไปยัง j
	*p = *p / 37   // หาร j ผ่านพอยน์เตอร์
	fmt.Println(j) // ดูค่าใหม่ของ j
}
