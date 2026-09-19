//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// এই মেথডটির অর্থ T টাইপ I ইন্টারফেস বাস্তবায়ন করে,
// তবে তা যে করে, সেটি স্পষ্টভাবে ডিক্লেয়ার করতে হয় না।
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
