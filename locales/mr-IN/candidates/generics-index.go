//go:build OMIT

package main

import "fmt"

// s मध्ये x चा इंडेक्स Index परत करते; x न सापडल्यास -1 देते.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v आणि x हे T टाइपचे आहेत; त्यावर comparable
		// कन्स्ट्रेंट असल्यामुळे येथे == वापरता येते.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index int च्या स्लाइसवरही काम करते
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index string च्या स्लाइसवरही काम करते
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
