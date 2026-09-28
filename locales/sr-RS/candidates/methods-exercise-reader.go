//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Додајте методу Read([]byte) (int, error) типу MyReader.

func main() {
	reader.Validate(MyReader{})
}
