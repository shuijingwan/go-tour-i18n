//go:build OMIT

package main

// List ಎನ್ನುವುದು ಯಾವುದೇ ಟೈಪ್‌ನ ಮೌಲ್ಯಗಳನ್ನು ಹೊಂದಿರುವ
// ಏಕ-ಸಂಪರ್ಕಿತ ಲಿಂಕ್‌ಡ್ ಲಿಸ್ಟ್ ಅನ್ನು ಪ್ರತಿನಿಧಿಸುತ್ತದೆ.
type List[T any] struct {
	next *List[T]
	val  T
}

func main() {
}
