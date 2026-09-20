//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// اس میتھڈ کا مطلب ہے کہ ٹائپ T انٹرفیس I کو implement کرتی ہے،
// لیکن ہمیں یہ بات explicit طور پر declare کرنے کی ضرورت نہیں۔
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
