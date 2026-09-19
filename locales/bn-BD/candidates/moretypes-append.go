//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append nil স্লাইসেও কাজ করে।
	s = append(s, 0)
	printSlice(s)

	// প্রয়োজন অনুযায়ী স্লাইসটি বড় হয়।
	s = append(s, 1)
	printSlice(s)

	// একবারে একাধিক উপাদান যোগ করা যায়।
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
