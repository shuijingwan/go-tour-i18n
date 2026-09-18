//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Thêm phương thức Read([]byte) (int, error) vào MyReader.

func main() {
	reader.Validate(MyReader{})
}
