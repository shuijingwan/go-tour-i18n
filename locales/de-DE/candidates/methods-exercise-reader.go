//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader um eine Methode Read([]byte) (int, error) erweitern.

func main() {
	reader.Validate(MyReader{})
}
