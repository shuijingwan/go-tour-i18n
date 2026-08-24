//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i を指す
	fmt.Println(*p) // ポインタ経由で i を読み取る
	*p = 21         // ポインタ経由で i を設定する
	fmt.Println(i)  // i の新しい値を確認する

	p = &j         // j を指す
	*p = *p / 37   // ポインタ経由で j を除算する
	fmt.Println(j) // j の新しい値を確認する
}
