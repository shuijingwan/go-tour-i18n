//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci تابعی است که
// تابعی را برمی‌گرداند که یک int برمی‌گرداند.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
