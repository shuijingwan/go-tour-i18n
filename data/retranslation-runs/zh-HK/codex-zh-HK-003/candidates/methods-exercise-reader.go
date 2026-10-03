//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// 待辦：為 MyReader 加入 Read([]byte) (int, error) 方法。

func main() {
	reader.Validate(MyReader{})
}
