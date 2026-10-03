//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Füge MyReader eine Methode Read([]byte) (int, error) hinzu.

func main() {
	reader.Validate(MyReader{})
}
