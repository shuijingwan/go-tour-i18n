//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // 指向 i
	fmt.Println(*p) // 透過指標讀取 i
	*p = 21         // 透過指標設定 i
	fmt.Println(i)  // 查看 i 的新值

	p = &j         // 指向 j
	*p = *p / 37   // 透過指標將 j 相除
	fmt.Println(j) // 查看 j 的新值
}
