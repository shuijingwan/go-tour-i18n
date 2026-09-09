//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader türüne bir Read([]byte) (int, error) metodu ekleyin.

func main() {
	reader.Validate(MyReader{})
}
