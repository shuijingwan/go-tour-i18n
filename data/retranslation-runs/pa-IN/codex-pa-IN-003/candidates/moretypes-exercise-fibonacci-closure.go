//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci ਇੱਕ ਅਜਿਹਾ ਫੰਕਸ਼ਨ ਹੈ ਜੋ
// int ਵਾਪਸ ਕਰਨ ਵਾਲਾ ਫੰਕਸ਼ਨ ਵਾਪਸ ਕਰਦਾ ਹੈ।
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
