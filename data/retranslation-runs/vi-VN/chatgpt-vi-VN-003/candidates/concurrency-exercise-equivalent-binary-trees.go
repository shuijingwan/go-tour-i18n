//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk duyệt cây t và gửi tất cả giá trị
// từ cây vào channel ch.
func Walk(t *tree.Tree, ch chan int)

// Same xác định liệu hai cây
// t1 và t2 có chứa cùng các giá trị hay không.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
