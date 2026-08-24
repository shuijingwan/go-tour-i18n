//go:build OMIT

package main

import "fmt"

// Index は s 内の x のインデックスを返します。見つからない場合は -1 を返します。
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v と x は comparable
		// 制約を持つ型 T なので、ここでは == を使用できます。
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index は int のスライスで動作します
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index は string のスライスでも動作します
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
