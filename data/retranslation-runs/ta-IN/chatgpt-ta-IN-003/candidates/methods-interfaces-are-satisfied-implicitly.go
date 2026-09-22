//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// இந்த வழிமுறை T வகை I இடைமுகத்தை நடைமுறைப்படுத்துகிறது என்பதைக் குறிக்கிறது,
// ஆனால் அது அப்படிச் செய்கிறது என்பதை நாம் வெளிப்படையாக அறிவிக்க வேண்டியதில்லை.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
