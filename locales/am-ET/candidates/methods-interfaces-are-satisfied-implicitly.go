//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// ይህ ዘዴ ዓይነት T በይነገጽ I ን እንደሚተገብር ያሳያል፣
// ነገር ግን ይህን በግልጽ መግለጽ አያስፈልገንም።
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
