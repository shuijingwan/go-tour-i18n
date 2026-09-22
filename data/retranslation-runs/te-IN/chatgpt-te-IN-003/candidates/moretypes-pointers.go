//go:build OMIT

package main

import "fmt"

func main() {
	i, j := 42, 2701

	p := &i         // i ను సూచించండి
	fmt.Println(*p) // పాయింటర్ ద్వారా i ను చదవండి
	*p = 21         // పాయింటర్ ద్వారా i ను సెట్ చేయండి
	fmt.Println(i)  // i యొక్క కొత్త విలువను చూడండి

	p = &j         // j ను సూచించండి
	*p = *p / 37   // పాయింటర్ ద్వారా j ను భాగించండి
	fmt.Println(j) // j యొక్క కొత్త విలువను చూడండి
}
