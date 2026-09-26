//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader ಗೆ Read([]byte) (int, error) ಮೆಥಡ್ ಅನ್ನು ಸೇರಿಸಿ.

func main() {
	reader.Validate(MyReader{})
}
