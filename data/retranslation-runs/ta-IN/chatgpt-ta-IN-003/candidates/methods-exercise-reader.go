//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: Read([]byte) (int, error) வழிமுறையை MyReader-க்கு சேர்க்கவும்.

func main() {
	reader.Validate(MyReader{})
}
