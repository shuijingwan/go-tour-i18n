//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Magdagdag ng method na Read([]byte) (int, error) sa MyReader.

func main() {
	reader.Validate(MyReader{})
}
