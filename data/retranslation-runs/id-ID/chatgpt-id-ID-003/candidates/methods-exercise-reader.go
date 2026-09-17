//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Tambahkan method Read([]byte) (int, error) ke MyReader.

func main() {
	reader.Validate(MyReader{})
}
