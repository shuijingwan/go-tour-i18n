//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// ЗАВДАННЯ: Додайте до MyReader метод Read([]byte) (int, error).

func main() {
	reader.Validate(MyReader{})
}
