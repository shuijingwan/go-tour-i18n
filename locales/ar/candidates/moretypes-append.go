//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// تعمل append على شرائح nil.
	s = append(s, 0)
	printSlice(s)

	// تكبر الشريحة حسب الحاجة.
	s = append(s, 1)
	printSlice(s)

	// يمكننا إضافة أكثر من عنصر واحد في المرة الواحدة.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
