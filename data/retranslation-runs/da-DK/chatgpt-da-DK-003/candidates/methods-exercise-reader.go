//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Tilføj en Read([]byte) (int, error)-metode til MyReader.

func main() {
	reader.Validate(MyReader{})
}
