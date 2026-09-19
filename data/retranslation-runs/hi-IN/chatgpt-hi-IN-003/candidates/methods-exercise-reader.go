//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader में Read([]byte) (int, error) मेथड जोड़ें।

func main() {
	reader.Validate(MyReader{})
}
