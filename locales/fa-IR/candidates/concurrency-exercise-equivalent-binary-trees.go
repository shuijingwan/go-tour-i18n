//go:build nobuild || OMIT

package main

import "golang.org/x/tour/tree"

// Walk درخت t را پیمایش می‌کند و همهٔ مقادیر را
// از درخت به کانال ch می‌فرستد.
func Walk(t *tree.Tree, ch chan int)

// Same مشخص می‌کند که آیا درخت‌های
// t1 و t2 مقادیر یکسانی دارند یا نه.
func Same(t1, t2 *tree.Tree) bool

func main() {
}
