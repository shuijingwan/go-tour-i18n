//go:build OMIT

package main

import "fmt"

// تعيد Index فهرس x في s، أو -1 إذا لم يُعثر عليه.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// كل من v وx من النوع T، الذي يحقق قيد النوع comparable،
		// لذلك يمكننا استخدام == هنا.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// تعمل Index على شريحة من قيم int
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// وتعمل Index أيضًا على شريحة من السلاسل النصية
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
