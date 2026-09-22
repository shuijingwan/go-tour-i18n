//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader కు Read([]byte) (int, error) మెథడ్‌ను జోడించండి.

func main() {
	reader.Validate(MyReader{})
}
