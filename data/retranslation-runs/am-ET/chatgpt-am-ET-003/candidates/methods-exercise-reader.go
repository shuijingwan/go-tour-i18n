//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: ወደ MyReader የ Read([]byte) (int, error) ዘዴ ጨምር።

func main() {
	reader.Validate(MyReader{})
}
