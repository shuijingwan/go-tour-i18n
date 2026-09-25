//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Добавете метод Read([]byte) (int, error) към MyReader.

func main() {
	reader.Validate(MyReader{})
}
