//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader માં Read([]byte) (int, error) મેથડ ઉમેરો.

func main() {
	reader.Validate(MyReader{})
}
