//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader میں Read([]byte) (int, error) میتھڈ شامل کریں۔

func main() {
	reader.Validate(MyReader{})
}
