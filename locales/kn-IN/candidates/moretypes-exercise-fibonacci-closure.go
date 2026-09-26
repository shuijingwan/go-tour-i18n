//go:build nobuild || OMIT

package main

import "fmt"

// fibonacci ಒಂದು ಫಂಕ್ಷನ್ ಆಗಿದ್ದು, ಅದು
// int ಮೌಲ್ಯವನ್ನು ಹಿಂತಿರುಗಿಸುವ ಮತ್ತೊಂದು ಫಂಕ್ಷನ್ ಅನ್ನು ಹಿಂತಿರುಗಿಸುತ್ತದೆ.
func fibonacci() func() int {
}

func main() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}
