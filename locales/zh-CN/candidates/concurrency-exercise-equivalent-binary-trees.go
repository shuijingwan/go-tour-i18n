//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk 遍历树 t，将其中的所有值
// 发送到通道 ch。
func Walk(t *tree.Tree, ch chan int)

// Same 判断两棵树
// t1 和 t2 是否包含相同的值。
func Same(t1, t2 *tree.Tree) bool

func main() {
}
