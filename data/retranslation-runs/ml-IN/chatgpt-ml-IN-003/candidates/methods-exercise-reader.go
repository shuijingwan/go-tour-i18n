//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader-ലേക്ക് Read([]byte) (int, error) മെഥഡ് ചേർക്കുക.

func main() {
	reader.Validate(MyReader{})
}
