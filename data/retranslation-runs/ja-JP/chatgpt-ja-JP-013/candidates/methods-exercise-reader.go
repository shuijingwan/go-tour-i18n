//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader に Read([]byte) (int, error) メソッドを追加してください。

func main() {
	reader.Validate(MyReader{})
}
