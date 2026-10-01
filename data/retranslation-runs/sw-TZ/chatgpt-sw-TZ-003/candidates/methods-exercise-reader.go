//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Ongeza mbinu ya Read([]byte) (int, error) kwenye MyReader.

func main() {
	reader.Validate(MyReader{})
}
