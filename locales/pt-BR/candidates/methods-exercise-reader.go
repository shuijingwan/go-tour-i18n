//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Adicione um método Read([]byte) (int, error) a MyReader.

func main() {
	reader.Validate(MyReader{})
}
