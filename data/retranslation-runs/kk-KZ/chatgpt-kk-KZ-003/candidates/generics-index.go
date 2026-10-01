//go:build OMIT

package main

import "fmt"

// Index s ішінен x индексін қайтарады, ал табылмаса -1 қайтарады.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v және x — comparable
		// шектеуі бар T түрінің мәндері, сондықтан мұнда == қолдана аламыз.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index бүтін сандар слайсымен жұмыс істейді
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index жолдар слайсымен де жұмыс істейді
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
