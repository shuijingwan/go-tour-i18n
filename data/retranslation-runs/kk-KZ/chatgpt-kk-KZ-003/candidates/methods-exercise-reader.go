//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader түріне Read([]byte) (int, error) әдісін қосыңыз.

func main() {
	reader.Validate(MyReader{})
}
