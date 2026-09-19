//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader-এ একটি Read([]byte) (int, error) মেথড যোগ করুন।

func main() {
	reader.Validate(MyReader{})
}
