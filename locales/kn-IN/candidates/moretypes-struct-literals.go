//go:build OMIT

package main

import "fmt"

type Vertex struct {
	X, Y int
}

var (
	v1 = Vertex{1, 2}  // Vertex ಟೈಪ್ ಅನ್ನು ಹೊಂದಿದೆ
	v2 = Vertex{X: 1}  // Y:0 ಅನ್ನು ಸ್ಪಷ್ಟವಾಗಿ ಸೂಚಿಸದಿದ್ದರೂ ಅದರ ಮೌಲ್ಯ 0 ಆಗಿರುತ್ತದೆ
	v3 = Vertex{}      // X:0 ಮತ್ತು Y:0
	p  = &Vertex{1, 2} // *Vertex ಟೈಪ್ ಅನ್ನು ಹೊಂದಿದೆ
)

func main() {
	fmt.Println(v1, p, v2, v3)
}
