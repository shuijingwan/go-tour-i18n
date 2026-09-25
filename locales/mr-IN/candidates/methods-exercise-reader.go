//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader मध्ये Read([]byte) (int, error) मेथड जोडा.

func main() {
	reader.Validate(MyReader{})
}
