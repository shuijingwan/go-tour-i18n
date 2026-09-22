//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// ఈ మెథడ్ అంటే టైప్ T, ఇంటర్‌ఫేస్ I ను అమలు చేస్తుందని అర్థం,
// కానీ అది అలా చేస్తుందని మనం స్పష్టంగా ప్రకటించాల్సిన అవసరం లేదు.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
