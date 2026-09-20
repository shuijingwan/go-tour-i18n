//go:build OMIT

package main

import "fmt"

// Index، s میں x کا index واپس کرتا ہے، یا نہ ملنے پر -1۔
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v اور x ٹائپ T کے ہیں، جس پر comparable
		// کنسٹرینٹ ہے، اس لیے یہاں == استعمال کر سکتے ہیں۔
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index، ints کی سلائس پر کام کرتا ہے
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index، strings کی سلائس پر بھی کام کرتا ہے
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
