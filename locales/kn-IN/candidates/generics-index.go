//go:build OMIT

package main

import "fmt"

// Index ಫಂಕ್ಷನ್ s ಸ್ಲೈಸ್‌ನಲ್ಲಿ x ನ ಇಂಡೆಕ್ಸ್ ಅನ್ನು ಹಿಂತಿರುಗಿಸುತ್ತದೆ; ಅದು ಸಿಗದಿದ್ದರೆ -1 ಅನ್ನು ಹಿಂತಿರುಗಿಸುತ್ತದೆ.
func Index[T comparable](s []T, x T) int {
	for i, v := range s {
		// v ಮತ್ತು x ಗಳು comparable ನಿರ್ಬಂಧವನ್ನು ಹೊಂದಿರುವ T ಟೈಪ್‌ನ ಮೌಲ್ಯಗಳು;
		// ಆದ್ದರಿಂದ ಇಲ್ಲಿ == ಅನ್ನು ಬಳಸಬಹುದು.
		if v == x {
			return i
		}
	}
	return -1
}

func main() {
	// Index ಪೂರ್ಣಾಂಕಗಳ ಸ್ಲೈಸ್‌ನಲ್ಲಿಯೂ ಕೆಲಸ ಮಾಡುತ್ತದೆ.
	si := []int{10, 20, 15, -10}
	fmt.Println(Index(si, 15))

	// Index ಸ್ಟ್ರಿಂಗ್‌ಗಳ ಸ್ಲೈಸ್‌ನಲ್ಲಿಯೂ ಕೆಲಸ ಮಾಡುತ್ತದೆ.
	ss := []string{"foo", "bar", "baz"}
	fmt.Println(Index(ss, "hello"))
}
