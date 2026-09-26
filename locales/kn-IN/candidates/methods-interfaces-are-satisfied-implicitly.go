//go:build OMIT

package main

import "fmt"

type I interface {
	M()
}

type T struct {
	S string
}

// ಈ ಮೆಥಡ್ ಇರುವುದರಿಂದ T ಟೈಪ್ I ಇಂಟರ್‌ಫೇಸ್ ಅನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುತ್ತದೆ;
// ಆದರೆ ಅದು ಹೀಗೆ ಅನುಷ್ಠಾನಗೊಳಿಸುತ್ತದೆ ಎಂದು ಪ್ರತ್ಯೇಕವಾಗಿ ಘೋಷಿಸುವ ಅಗತ್ಯವಿಲ್ಲ.
func (t T) M() {
	fmt.Println(t.S)
}

func main() {
	var i I = T{"hello"}
	i.M()
}
