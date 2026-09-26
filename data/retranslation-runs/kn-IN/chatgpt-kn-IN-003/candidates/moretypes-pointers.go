//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i ಅನ್ನು ಸೂಚಿಸಿ
	fmt.Println(*p) // ಪಾಯಿಂಟರ್ ಮೂಲಕ i ಮೌಲ್ಯವನ್ನು ಓದಿ
	*p = 21         // ಪಾಯಿಂಟರ್ ಮೂಲಕ i ಗೆ ಮೌಲ್ಯ ನೀಡಿ
	fmt.Println(i)  // i ನ ಹೊಸ ಮೌಲ್ಯವನ್ನು ನೋಡಿ

	p = &j         // j ಅನ್ನು ಸೂಚಿಸಿ
	*p = *p / 37   // ಪಾಯಿಂಟರ್ ಮೂಲಕ j ಅನ್ನು ಭಾಗಿಸಿ
	fmt.Println(j) // j ನ ಹೊಸ ಮೌಲ್ಯವನ್ನು ನೋಡಿ
}
