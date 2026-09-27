//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Legg til metoden Read([]byte) (int, error) i MyReader.

func main() {
	reader.Validate(MyReader{})
}
