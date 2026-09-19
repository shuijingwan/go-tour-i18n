//go:build OMIT

package main

import "fmt"

// Index s-এ x-এর ইনডেক্স রিটার্ন করে, না পেলে -1 রিটার্ন করে।
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v এবং x হলো T টাইপের, যার comparable
		// টাইপ সীমাবদ্ধতা আছে, তাই এখানে == ব্যবহার করা যায়।
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index পূর্ণসংখ্যার একটি স্লাইসে কাজ করে
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index স্ট্রিংয়ের একটি স্লাইসেও কাজ করে
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
