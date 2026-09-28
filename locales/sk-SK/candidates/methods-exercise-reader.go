//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Pridaj metódu Read([]byte) (int, error) do MyReader.

func main() {
	reader.Validate(MyReader{})
}
