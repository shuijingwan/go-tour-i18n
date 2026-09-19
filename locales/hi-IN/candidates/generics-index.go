//go:build OMIT

package main

import "fmt"

// Index, s में x का इंडेक्स लौटाता है, या न मिलने पर -1।
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v और x टाइप T के हैं, जिस पर comparable
		// कंस्ट्रेंट लागू है, इसलिए यहाँ == का उपयोग कर सकते हैं।
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index, int की स्लाइस पर काम करता है
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index, string की स्लाइस पर भी काम करता है
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
