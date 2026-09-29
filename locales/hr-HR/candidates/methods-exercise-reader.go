//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Dodaj metodu Read([]byte) (int, error) tipu MyReader.

func main() {
	reader.Validate(MyReader{})
}
