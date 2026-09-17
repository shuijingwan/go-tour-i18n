//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO：為 MyReader 新增 Read([]byte) (int, error) 方法。

func main() {
	reader.Validate(MyReader{})
}
