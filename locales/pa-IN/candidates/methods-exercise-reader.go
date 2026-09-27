//go:build nobuild || OMIT

package main

import "golang.org/x/tour/reader"

type MyReader struct{}

// TODO: MyReader ਵਿੱਚ ਇੱਕ Read([]byte) (int, error) ਮੈਥਡ ਜੋੜੋ।

func main() {
	reader.Validate(MyReader{})
}
