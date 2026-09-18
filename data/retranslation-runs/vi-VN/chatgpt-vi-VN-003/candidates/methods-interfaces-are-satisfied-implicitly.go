//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Phương thức này có nghĩa là kiểu T triển khai interface I,
// nhưng ta không cần khai báo tường minh rằng nó làm như vậy.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
