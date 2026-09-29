//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: pievienot MyReader metodi Read([]byte) (int, error).

func main() {
	reader.Validate(MyReader{})
}
