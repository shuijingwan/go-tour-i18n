//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Adj a MyReader típushoz egy Read([]byte) (int, error) metódust.

func main() {
	reader.Validate(MyReader{})
}
