//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: أضف تابع Read([]byte) (int, error) إلى MyReader.

func main() {
	reader.Validate(MyReader{})
}
