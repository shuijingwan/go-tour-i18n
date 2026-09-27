//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// ਇਸ ਮੈਥਡ ਦਾ ਮਤਲਬ ਹੈ ਕਿ ਟਾਈਪ T, ਇੰਟਰਫੇਸ I ਨੂੰ ਲਾਗੂ ਕਰਦੀ ਹੈ,
// ਪਰ ਸਾਨੂੰ ਇਸ ਗੱਲ ਦੀ ਸਪਸ਼ਟ ਘੋਸ਼ਣਾ ਕਰਨ ਦੀ ਲੋੜ ਨਹੀਂ ਹੈ।
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
