//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Eine Methode Read([]byte) (int, error) zu MyReader hinzufügen.

func main() {
	reader.Validate(MyReader{})
}
