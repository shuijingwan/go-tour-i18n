//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk 走訪樹 t，將樹中的所有值
// 傳送到通道 ch。
func Walk(t *tree.Tree, ch chan int)

// Same 判斷兩棵樹
// t1 與 t2 是否包含相同的值。
func Same(t1, t2 *tree.Tree) bool

func main() {
}
