//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Προσθέστε μια μέθοδο Read([]byte) (int, error) στον MyReader.

func main() {
	reader.Validate(MyReader{})
}
