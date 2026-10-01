//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: یک متد Read([]byte) (int, error) به MyReader اضافه کنید.

func main() {
	reader.Validate(MyReader{})
}
