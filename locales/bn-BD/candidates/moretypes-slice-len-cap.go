//go:build OMIT

package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11, 13}
	printSlice(s)

	// স্লাইসটিকে এমনভাবে স্লাইস করুন যাতে এর দৈর্ঘ্য শূন্য হয়।
	s = s[:0]
	printSlice(s)

	// এর দৈর্ঘ্য বাড়ান।
	s = s[:4]
	printSlice(s)

	// এর প্রথম দুটি মান বাদ দিন।
	s = s[2:]
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
