//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader에 Read([]byte) (int, error) 메서드를 추가하세요.

func main() {
	reader.Validate(MyReader{})
}
