//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append פועלת גם על פרוסות nil.
	s = append(s, 0)
	printSlice(s)

	// הפרוסה גדלה לפי הצורך.
	s = append(s, 1)
	printSlice(s)

	// אפשר להוסיף יותר מאיבר אחד בכל פעם.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
