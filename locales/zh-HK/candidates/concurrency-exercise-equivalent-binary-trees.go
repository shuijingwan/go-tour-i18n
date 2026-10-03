//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk 會遍歷樹 t，並將所有值
// 從樹傳送至通道 ch。
func Walk(t *tree.Tree, ch chan int)

// Same 會判斷兩棵樹
// t1 和 t2 是否包含相同的值。
func Same(t1, t2 *tree.Tree) bool

func main() {
}
