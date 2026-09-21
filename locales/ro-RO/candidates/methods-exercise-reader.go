//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Adaugă o metodă Read([]byte) (int, error) la MyReader.

func main() {
	reader.Validate(MyReader{})
}
