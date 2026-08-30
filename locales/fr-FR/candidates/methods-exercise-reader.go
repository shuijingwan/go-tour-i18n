//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: ajoutez une méthode Read([]byte) (int, error) à MyReader.

func main() {
	reader.Validate(MyReader{})
}
