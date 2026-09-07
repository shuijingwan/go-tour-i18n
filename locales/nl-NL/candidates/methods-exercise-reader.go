//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Voeg een methode Read([]byte) (int, error) toe aan MyReader.

func main() {
	reader.Validate(MyReader{})
}
