//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// Αυτή η μέθοδος σημαίνει ότι ο τύπος T υλοποιεί τη διεπαφή I,
// αλλά δεν χρειάζεται να δηλώσουμε ρητά ότι την υλοποιεί.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
