//go:build OMIT

package main

import "fmt"

func main() {
	var s []int
	printSlice(s)

	// append روی اسلایس‌های nil کار می‌کند.
	s = append(s, 0)
	printSlice(s)

	// اسلایس به‌اندازهٔ نیاز رشد می‌کند.
	s = append(s, 1)
	printSlice(s)

	// می‌توانیم هر بار بیش از یک عنصر اضافه کنیم.
	s = append(s, 2, 3, 4)
	printSlice(s)
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
