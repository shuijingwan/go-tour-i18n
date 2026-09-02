//go:build OMIT

package main

import "fmt"

// Index는 s에서 x의 인덱스를 반환하며, 찾지 못하면 -1을 반환합니다.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v와 x의 타입은 T이며, T에는 comparable
		// 타입 제약이 있으므로 여기서 ==를 사용할 수 있습니다.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index는 int 슬라이스에 사용할 수 있습니다
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index는 string 슬라이스에도 사용할 수 있습니다
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
