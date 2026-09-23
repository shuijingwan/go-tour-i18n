//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Tambahkan kaedah Read([]byte) (int, error) kepada MyReader.

func main() {
	reader.Validate(MyReader{})
}
