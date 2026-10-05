//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// צרו מהפרוסה פרוסה באורך אפס.
	s = s[:0]
	printSlice(s)

	// האריכו אותה.
	s = s[:4]
	printSlice(s)

	// השמיטו את שני הערכים הראשונים שלה.
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
