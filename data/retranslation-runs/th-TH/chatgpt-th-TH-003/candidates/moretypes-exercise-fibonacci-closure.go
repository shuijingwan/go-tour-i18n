//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci คือฟังก์ชันที่คืน
// ฟังก์ชันที่คืนค่า int
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
