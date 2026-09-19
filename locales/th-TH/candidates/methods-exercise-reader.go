//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: เพิ่มเมธอด Read([]byte) (int, error) ให้ MyReader

func main() {
	reader.Validate(MyReader{})
}
