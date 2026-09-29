//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Lisa tüübile MyReader meetod Read([]byte) (int, error).

func main() {
	reader.Validate(MyReader{})
}
