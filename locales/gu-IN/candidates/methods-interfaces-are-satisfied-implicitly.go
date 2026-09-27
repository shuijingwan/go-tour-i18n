//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// આ મેથડનો અર્થ એ છે કે ટાઇપ T, ઇન્ટરફેસ I નું અમલીકરણ કરે છે,
// પણ એવું કરે છે તેવી સ્પષ્ટ ઘોષણા કરવાની જરૂર નથી.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
