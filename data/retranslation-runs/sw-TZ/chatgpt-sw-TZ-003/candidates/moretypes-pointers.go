//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // elekeza kwa i
	fmt.Println(*p) // soma i kupitia kielekezi
	*p = 21         // weka i kupitia kielekezi
	fmt.Println(i)  // ona thamani mpya ya i

	p = &j         // elekeza kwa j
	*p = *p / 37   // gawanya j kupitia kielekezi
	fmt.Println(j) // ona thamani mpya ya j
}
